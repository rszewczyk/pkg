package http

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type callChain struct {
	chain []int
}

// wraps a Handler and keeps track of the order in which it was called
type calledHandler struct {
	http.Handler
	c    *callChain
	step int
}

func (h *calledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.c.chain = append(h.c.chain, h.step)
	h.Handler.ServeHTTP(w, r)
}

type calledMuxer struct {
	H http.Handler
	Muxable
	c    *callChain
	step int
}

func (m *calledMuxer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.c.chain = append(m.c.chain, m.step)
	m.H.ServeHTTP(w, r)
}

func runHandlerTest(t *testing.T, makeHandler func(int) http.Handler, expectedChain []int, w http.ResponseWriter, r *http.Request) {
	actualChain := &callChain{[]int{}}

	handlers := []http.Handler{}
	for i := 0; i < 10; i++ {
		h := makeHandler(i)
		if m, ok := h.(Muxable); ok {
			handlers = append(handlers, &calledMuxer{h, m, actualChain, i})
		} else {
			handlers = append(handlers, &calledHandler{h, actualChain, i})
		}
	}

	NewHandlerChain(handlers...).ServeHTTP(w, r)
	actualLength, expectedLength := len(actualChain.chain), len(expectedChain)
	if actualLength != expectedLength {
		t.Fatal("Expected a call chain with length", expectedLength, "Got", actualLength)
	}

	for i := range actualChain.chain {
		if actualChain.chain[i] != expectedChain[i] {
			t.Fatal("Expected call chain to be", expectedChain, "Got", actualChain)
		}
	}
}

func TestHandlerChainCallOrder(t *testing.T) {
	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	}
	runHandlerTest(t, makeHandler, []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestHandlerChainTerminatesOnWrite(t *testing.T) {
	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case step == 5:
				w.Write([]byte("some content"))
			case step >= 6:
				t.Errorf("Did not expect to make it beyond step 5 in the chain")
			}
		})
	}
	runHandlerTest(t, makeHandler, []int{0, 1, 2, 3, 4, 5}, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestHandlerChainTerminatesOnWriteHeader(t *testing.T) {
	makeHandler := func(step int) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case step == 4:
				w.WriteHeader(200)
			case step >= 5:
				t.Errorf("Did not expect to make it beyond step 4 in the chain")
			}
		})
	}
	runHandlerTest(t, makeHandler, []int{0, 1, 2, 3, 4}, &httptest.ResponseRecorder{}, &http.Request{})
}

func TestHandlerChainProcessesServeMux(t *testing.T) {
	r := &http.Request{
		URL: &url.URL{
			Path: "/foobar",
		},
	}

	makeHandler := func(step int) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {}

		switch {
		case step == 2:
			//since there's no registered handler for foobar, the chain should skip here
			return http.NewServeMux()
		case step == 4:
			m := http.NewServeMux()
			m.HandleFunc("/foobar", fn)
			return m
		}

		return http.HandlerFunc(fn)
	}
	runHandlerTest(t, makeHandler, []int{0, 1, 3, 4, 5, 6, 7, 8, 9}, &httptest.ResponseRecorder{}, r)
}
