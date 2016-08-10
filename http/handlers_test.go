package http

import (
	"context"
	"net/http"
	"net/http/httptest"
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

func intSlicesAreEqual(first, second []int) bool {
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

	expectedDeadline := time.Now().Add(10 * time.Minute)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if actualDeadline, ok := r.Context().Deadline(); !ok || expectedDeadline != actualDeadline {
			t.Errorf("%s: expected deadline to be %v, got %v", testName, expectedDeadline, actualDeadline)
		}
	})

	r := &http.Request{Header: make(map[string][]string)}
	r.Header.Set(DeadlineHeaderKey, expectedDeadline.Format(time.RFC3339Nano))

	DeadlineHandler(h).ServeHTTP(httptest.NewRecorder(), r)
}

func TestDeadlineHandlerWithoutHeader(t *testing.T) {
	testName := "TestDeadlineHandlerWithoutHeader"

	var expected context.Context
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if actual := r.Context(); expected != actual {
			t.Errorf("%s: Expected context to be %v, got %v", testName, expected, actual)
		}
	})

	r := &http.Request{}
	expected = r.Context()

	DeadlineHandler(h).ServeHTTP(httptest.NewRecorder(), r)
}

func TestComposeHandlerWrappers(t *testing.T) {
	testName := "TestComposeHandlerWrappers"

	var (
		expectedOrder []int
		wrappers      []HandlerWrapper
		wasCalled     bool
	)
	actualOrder := new(callOrder)
	for i := 0; i < 10; i++ {
		expectedOrder = append(expectedOrder, i)
		wrappers = append(wrappers, actualOrder.wrapper(i))
	}

	// wrapper handler should be last
	expectedOrder = append(expectedOrder, 11)

	ComposeHandlerWrappers(wrappers...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wasCalled = true
		*actualOrder = append(*actualOrder, 11)
	})).ServeHTTP(httptest.NewRecorder(), &http.Request{})

	if !wasCalled {
		t.Error(testName + " (1): wrapped handler was not called")
	}
	if !intSlicesAreEqual(expectedOrder, *actualOrder) {
		t.Errorf("%s: (2): Expected call order was %v, got %v", testName, expectedOrder, *actualOrder)
	}
}
