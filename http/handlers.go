package http

import (
	"context"
	"net/http"
	"strconv"
	"time"
)

// DeadlineHeaderKey is used as the key for the Deadline header
const DeadlineHeaderKey = "Deadline"

var unixEpoch = time.Date(1970, time.January, 0, 0, 0, 0, 0, time.UTC)

func writeErr(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

// AllowedHandler returns an http.Handler that will reply with a status code of
// 405 and write the appropriate Allowed header if the request method is not one
// of the allowed methods.
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

// DeadlineHandler adds a deadline to the context of the request that is passed to next.ServeHTTP. The
// value of the deadline will be determined via the request header called "Deadline", the first value of
// which must be parseable as a 64 bit integer and must represent the duration in seconds since the Unix
// epoch (Jan 1, 1970). If a header value is present, but cannot be parsed the DeadlineHandler will respond
// with http.StatusBadRequest.
func DeadlineHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hdr := r.Header.Get(DeadlineHeaderKey)
		if hdr == "" {
			next.ServeHTTP(w, r)
			return
		}

		deadline, err := strconv.ParseInt(hdr, 10, 64)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusBadRequest)+": invalid deadline "+hdr, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithDeadline(r.Context(), unixEpoch.Add(time.Second*time.Duration(deadline)))
		defer cancel()
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
