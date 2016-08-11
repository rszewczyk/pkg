package http

import (
	"context"
	"net/http"
	"time"
)

// DeadlineHeaderKey is used as the key for the Deadline header
const DeadlineHeaderKey = "Deadline"

func writeErr(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

// AllowedHandler returns an http.Handler that will reply with a status code of
// 405 and write the appropriate Allowed header if the request method is not one
// of the allowed methods
func AllowedHandler(allowed ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, m := range allowed {
			if m == r.Method {
				return
			}
		}
		w.Header().Del("Allowed")
		for _, m := range allowed {
			w.Header().Add("Allowed", m)
		}
		writeErr(w, http.StatusMethodNotAllowed)
	})
}

// LengthRequiredHandler will reply with status 411 if the length of the request body is unknown.
func LengthRequiredHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.ContentLength < 0 {
			writeErr(w, http.StatusLengthRequired)
		}
	})
}

// DeadlineHandler wraps h in an http.Handler that adds a deadline to the request Context. The value
// of the deadline will be determined via a request header called "Deadline" the value of which
// should be compliant with RFC3339Nano
func DeadlineHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rNew := r
		deadline, err := time.Parse(time.RFC3339Nano, r.Header.Get(DeadlineHeaderKey))
		if err == nil {
			ctx, cancel := context.WithDeadline(r.Context(), deadline)
			rNew = r.WithContext(ctx)
			go func() {
				<-ctx.Done()
				cancel()
			}()
		}
		h.ServeHTTP(w, rNew)
	})
}
