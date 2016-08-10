package http

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

func (c *callOrder) wrapper(step int) HandlerWrapper {
	return func(h http.Handler) http.Handler {
		return c.wrap(step, h)
	}
}

func runHandlerChainTest(t *testing.T, testName string, makeHandler func(int) http.Handler, expectedOrder []int, w http.ResponseWriter, r *http.Request) {
	actualOrder := new(callOrder)

	handlers := []http.Handler{}
	for i := 0; i < 10; i++ {
		handlers = append(handlers, actualOrder.wrap(i, makeHandler(i)))
	}

	NewHandlerChain(handlers...).ServeHTTP(w, r)

	if !intSlicesAreEqual(expectedOrder, *actualOrder) {
		t.Errorf("%s: expected call order %v, got %v", testName, expectedOrder, *actualOrder)
	}
}

func TestHandlerChainCallOrder(t *testing.T) {
	testName := "TestHandlerChainCallOrder"
	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	}
	runHandlerChainTest(t, testName, makeHandler, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestHandlerChainTerminatesOnWrite(t *testing.T) {
	testName := "TestHandlerChainTerminatesOnWrite"

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
	runHandlerChainTest(t, testName, makeHandler, []int{0, 1, 2, 3, 4, 5}, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestHandlerChainTerminatesOnWriteHeader(t *testing.T) {
	testName := "TestHandlerChainTerminatesOnWriteHeader"

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
	runHandlerChainTest(t, testName, makeHandler, []int{0, 1, 2, 3, 4}, &httptest.ResponseRecorder{}, &http.Request{})
}
