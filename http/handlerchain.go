package http

import "net/http"

// NewHandlerChain returns a handler that calls each of provided handlers in
// left to right order, stopping after the first handler that writes to the
// ResponseWriter. If a particular handler implements Muxable, as does http.ServeMux
// from the standard library, it will be queried to see if it has a handler
// registered for the requested URL. If it does not, it is skipped so as to
// avoid calling its default handler.
//
// A handler chain is an alternative to the oft seen middleware pattern where
// a next parameter is passed along with the Request and ResponseWriter. This
// allows direct composition of types that implement the standard http.Handler
// interface.
func NewHandlerChain(handlers ...http.Handler) http.Handler {
	return &handlerChain{handlers}
}

type handlerChain struct {
	handlers []http.Handler
}

// ServeHttp implements the http.Handler interface
func (h *handlerChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	checkable := &chainResponseWriter{ResponseWriter: w}

	// call each handler in turn, stopping when a response has been written
	for _, handler := range h.handlers {
		// if this is a mux like handler, skip if it doesn't apply
		if mux, ok := handler.(Muxable); ok {
			if _, p := mux.Handler(r); p == "" {
				continue
			}
		}

		handler.ServeHTTP(checkable, r)
		if checkable.written {
			return
		}
	}
}

// Muxable represents the behavior that allows a type such as http.ServeMux to
// be queried for a handler and pattern for a given request
type Muxable interface {
	// Handler returns the handler to use for the given request and registered pattern that matches the
	// request. If there is no registered handler that applies to the request, Handler returns an empty pattern
	// and a default handler.
	Handler(r *http.Request) (h http.Handler, pattern string)
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
