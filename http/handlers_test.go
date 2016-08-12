package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestLengthRequiredHandler(t *testing.T) {
	testName := "TestLengthRequiredHandler"

	h := LengthRequiredHandler()
	r := &http.Request{ContentLength: -1}
	w := httptest.NewRecorder()

	h.ServeHTTP(w, r)
	if expected, actual := http.StatusLengthRequired, w.Code; expected != actual {
		t.Errorf("%s (1): expected code to be %d, got %d", testName, expected, actual)
	}
	if expected, actual := http.StatusText(http.StatusLengthRequired), w.Body.String(); !strings.Contains(actual, expected) {
		t.Errorf("%s (2): expected body to contain '%s', got '%s'", testName, expected, actual)
	}
}

func TestAllowedHandler(t *testing.T) {
	testName := "TestAllowedHandler"

	// for testing a response that already has an Allowed header
	w := httptest.NewRecorder()
	w.HeaderMap = map[string][]string{"Allowed": []string{http.MethodPost}}

	tests := []struct {
		r       *http.Request
		w       *httptest.ResponseRecorder
		methods []string
	}{
		{&http.Request{Method: http.MethodDelete}, httptest.NewRecorder(), []string{http.MethodGet, http.MethodPut}},
		{&http.Request{Method: http.MethodPost}, httptest.NewRecorder(), []string{http.MethodPut}},
		{&http.Request{Method: http.MethodHead}, httptest.NewRecorder(), []string{http.MethodPost, http.MethodHead}},
		{&http.Request{Method: http.MethodPost}, w, []string{http.MethodDelete, http.MethodGet}},
	}

	for i, test := range tests {
		expectedStatus := http.StatusMethodNotAllowed
		for _, m := range test.methods {
			if test.r.Method == m {
				expectedStatus = test.w.Code
				break
			}
		}
		h := AllowedHandler(test.methods...)

		h.ServeHTTP(test.w, test.r)
		if actualStatus := test.w.Code; expectedStatus != actualStatus {
			t.Errorf("%s loop(%d) (1): Expected status to be %d, got %d", testName, i, expectedStatus, actualStatus)
		} else if expectedHeader, actualHeader := test.methods, test.w.HeaderMap[http.CanonicalHeaderKey("Allowed")]; actualStatus == http.StatusMethodNotAllowed && !stringSlicesAreEqual(expectedHeader, actualHeader) {
			t.Errorf("%s loop(%d) (2): Expected headers to be %v, got %v", testName, i, expectedHeader, actualHeader)
		}
	}
}

func TestDeadlineHandlerWithHeader(t *testing.T) {
	testName := "TestDeadlineHandlerWithHeader"

	handlerWasCalled := false
	expectedDeadline := time.Now().Add(10 * time.Minute)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerWasCalled = true
		if actualDeadline, ok := r.Context().Deadline(); !ok || expectedDeadline.Equal(actualDeadline) {
			t.Errorf("%s: expected deadline to be %s, got %s", testName, expectedDeadline.UTC(), actualDeadline.UTC())
		}
	})

	r := &http.Request{Header: make(map[string][]string)}
	r.Header.Set(DeadlineHeaderKey, strconv.FormatInt(expectedDeadline.Unix(), 10))

	DeadlineHandler(h).ServeHTTP(httptest.NewRecorder(), r)
	if !handlerWasCalled {
		t.Error(testName + ": Expected handler to have been called")
	}
}

func TestDeadlineHandlerWithoutHeader(t *testing.T) {
	testName := "TestDeadlineHandlerWithoutHeader"

	var expected context.Context
	handlerWasCalled := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerWasCalled = true
		if actual := r.Context(); expected != actual {
			t.Errorf("%s: Expected context to be %v, got %v", testName, expected, actual)
		}
	})

	r := &http.Request{}
	expected = r.Context()

	DeadlineHandler(h).ServeHTTP(httptest.NewRecorder(), r)
	if !handlerWasCalled {
		t.Error(testName + ": Expected handler to have been called")
	}
}

func TestDeadlineHandlerWithBadHeader(t *testing.T) {
	testName := "TestDeadlineHandlerWithBadHeader"

	handlerWasCalled := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerWasCalled = true
	})

	r := &http.Request{Header: make(map[string][]string)}
	r.Header.Set(DeadlineHeaderKey, "foobar")
	rec := httptest.NewRecorder()

	DeadlineHandler(h).ServeHTTP(rec, r)
	if handlerWasCalled {
		t.Error(testName + ": Did not expect handler to be called")
	}
	if expected, actual := http.StatusBadRequest, rec.Code; expected != actual {
		t.Errorf("%s: Expected code to be %d, got %d", testName, expected, actual)
	}
}

func stringSlicesAreEqual(first, second []string) bool {
	if first == nil && second == nil {
		return true
	}
	if len(first) != len(second) {
		return false
	}
	for i, s := range first {
		if second[i] != s {
			return false
		}
	}
	return true
}
