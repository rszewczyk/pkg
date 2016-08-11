/*
Package middleware provides utilities for composing types that implement http.Handler

A middleware is a function with the following signature:

	func(next http.Handler) http.Handler

The returned Handler either writes the response or calls the ServeHTTP method of the next Handler.

Example

	package main

	import (
	        "net/http"
	)

	func GetOnly(next http.Handler) http.Handler {
	        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	                if r.Method != http.MethodGet {
	                        w.Header().Set("Allowed", http.MethodGet)
	                        http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	                        return
	                }
	                next.ServeHTTP(w, r)
	        })
	}

	func main() {
	        http.ListenAndServe(":8000", GetOnly(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	                w.Write([]byte("Hello!\n"))
	        })))
	}
*/
package middleware

import "net/http"

// Compose returns a middleware that is a composition of the argument middlewares. The middlewares are composed in reverse
// argument order - i.e. the last argument is the innermost middleware.
func Compose(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(last http.Handler) (composition http.Handler) {
		composition = last
		for i := len(middlewares) - 1; i >= 0; i-- {
			composition = middlewares[i](composition)
		}
		return
	}
}

// AdaptHandler allows any http.Handler to act as a middleware. The returned middleware will check if the adapted
// Handler writes the response and call the next handler's ServeHTTP method if it doesn't.
func AdaptHandler(h http.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cw := &checkableWriter{ResponseWriter: w}
			h.ServeHTTP(cw, r)
			if cw.written {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Chain returns an http.Handler that will call the ServeHTTP method of each of the provided handlers in argument order.
// After calling each handler, a check will be made to see if the response has been written and if so terminate the chain.
func Chain(handlers ...http.Handler) http.Handler {
	return &handlerChain{handlers}
}

type handlerChain struct {
	handlers []http.Handler
}

func (h *handlerChain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cw := &checkableWriter{ResponseWriter: w}
	for _, h := range h.handlers {
		h.ServeHTTP(cw, r)
		if cw.written {
			return
		}
	}
}

type checkableWriter struct {
	http.ResponseWriter
	written bool
}

func (w *checkableWriter) Write(p []byte) (int, error) {
	w.written = true
	return w.ResponseWriter.Write(p)
}

func (w *checkableWriter) WriteHeader(code int) {
	w.written = true
	w.ResponseWriter.WriteHeader(code)
}
