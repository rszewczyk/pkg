package http

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPipeWriter(t *testing.T) {
	testName := "TestPipeWriter"

	expectedContent := []byte("some content")

	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		actualContent, _ := ioutil.ReadAll(r.Body)
		if bytes.Compare(expectedContent, actualContent) != 0 {
			t.Errorf("%s (1): Expected content to be '%s', got '%s'", testName, string(expectedContent), string(actualContent))
		}
		w.Write(expectedContent)
	}))
	defer svr.Close()

	resultCh := make(chan Result)
	req, _ := http.NewRequest(http.MethodPost, svr.URL, nil)

	w := PipeWriter(nil, req, resultCh)
	w.Write(expectedContent)
	w.Close()

	result := <-resultCh
	if result.Error != nil {
		t.Errorf("%s (2): Expected result.Error to be nil, got %s", testName, result.Error)
	}
	if result.Request != req {
		t.Error(testName + " (3): Expected requests to be the same")
	}

	defer result.Response.Body.Close()
	actualContent, _ := ioutil.ReadAll(result.Response.Body)
	if bytes.Compare(expectedContent, actualContent) != 0 {
		t.Errorf("%s (4): Expected content to be '%s', got '%s'", testName, string(expectedContent), string(actualContent))
	}
}
