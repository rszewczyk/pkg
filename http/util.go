package http

import (
	"io"
	"net/http"
)

// Result represents the result of executing an HTTP request. It holds any response and error
// obtained as well as the original request.
type Result struct {
	Response *http.Response
	Request  *http.Request
	Error    error
}

// PipeWriter allows an http request body to be streamed through a writer. It executes req using c (if
// c is nil then http.DefaultClient will be used). Callers must close w when finished writing the request
// body. The result will be placed on resultCh.
func PipeWriter(c *http.Client, req *http.Request, resultCh chan<- Result) (w io.WriteCloser) {
	if c == nil {
		c = http.DefaultClient
	}
	req.Body, w = io.Pipe()

	go func() {
		res, err := c.Do(req)
		resultCh <- Result{res, req, err}
	}()
	return
}
