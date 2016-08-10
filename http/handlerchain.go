package http

import "net/http"

// NewHandlerChain returns a handler that calls each of provided handlers in
// left to right order, stopping after the first handler that writes to the
// ResponseWriter.
func NewHandlerChain(handlers ...http.Handler) http.Handler {
	return &handlerChain{handlers}
}

type handlerChain struct {
	handlers []http.Handler
}

// ServeHTTP implements the http.Handler interface
func (h *handlerChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	crw := &chainResponseWriter{ResponseWriter: w}
	for _, h := range h.handlers {
		h.ServeHTTP(crw, r)
		if crw.written {
			return
		}
	}
}

type chainResponseWriter struct {
	http.ResponseWriter
	written bool
}

func (w *chainResponseWriter) Write(p []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(p)
}

func (w *chainResponseWriter) WriteHeader(code int) {
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}
