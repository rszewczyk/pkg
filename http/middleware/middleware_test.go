package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type callOrder []int

func (c *callOrder) wrap(step int, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*c = append(*c, step)
		h.ServeHTTP(w, r)
	})
}

func (c *callOrder) wrapper(step int) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return c.wrap(step, h)
	}
}

func TestCompose(t *testing.T) {
	testName := "TestCompose"

	var (
		expectedOrder []int
		actualOrder   = new(callOrder)
		middlewares   []func(http.Handler) http.Handler
		wasCalled     bool
	)

	for i := 0; i < 10; i++ {
		expectedOrder = append(expectedOrder, i)
		middlewares = append(middlewares, actualOrder.wrapper(i))
	}

	// the wrapped handler should be last
	expectedOrder = append(expectedOrder, 11)

	Compose(middlewares...)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

func TestAdaptHandlerNextNotCalledOnWrite(t *testing.T) {
	testName := "TestAdaptHandlerNext"

	AdaptHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error(testName + " (1): did not expect to reach the final handler")
	})).ServeHTTP(httptest.NewRecorder(), &http.Request{})
}

func TestAdaptHandlerNextIsCalled(t *testing.T) {
	testName := "TestAdaptHandlerNextIsCalled"

	nextCalled := false
	AdaptHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})).ServeHTTP(httptest.NewRecorder(), &http.Request{})

	if !nextCalled {
		t.Error(testName + " (1): expected the final handler to be called")
	}
}

func runChainTest(t *testing.T, testName string, makeHandler func(int) http.Handler, stepCount int, w http.ResponseWriter, r *http.Request) {
	var (
		actualOrder   = new(callOrder)
		expectedOrder []int
		handlers      []http.Handler
	)

	for i := 0; i < 10; i++ {
		handlers = append(handlers, actualOrder.wrap(i, makeHandler(i)))
		if i < stepCount {
			expectedOrder = append(expectedOrder, i)
		}
	}

	Chain(handlers...).ServeHTTP(w, r)

	if !intSlicesAreEqual(expectedOrder, *actualOrder) {
		t.Errorf("%s: expected call order %v, got %v", testName, expectedOrder, *actualOrder)
	}
}

func TestChainCallOrder(t *testing.T) {
	testName := "TestChainCallOrder"
	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	}
	runChainTest(t, testName, makeHandler, 10, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestChainTerminatesOnWrite(t *testing.T) {
	testName := "TestChainTerminatesOnWrite"

	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case step == 5:
				w.Write([]byte("some content"))
			case step >= 6:
				t.Errorf(testName + " did not expect to make it beyond step 5 in the chain")
			}
		})
	}
	runChainTest(t, testName, makeHandler, 6, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestChainTerminatesOnWriteHeader(t *testing.T) {
	testName := "TestChainTerminatesOnWriteHeader"

	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case step == 4:
				w.WriteHeader(200)
			case step >= 5:
				t.Error(testName + " Did not expect to make it beyond step 4 in the chain")
			}
		})
	}
	runChainTest(t, testName, makeHandler, 5, &httptest.ResponseRecorder{}, &http.Request{})
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
