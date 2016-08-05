package ioutil

import (
	"bytes"
	"errors"
	"io/ioutil"
	"testing"
)

func TestReadCloser(t *testing.T) {
	r := CallbackReadCloser(nil, nil)

	if actual := r.Close(); actual != nil {
		t.Error("Expected a nil error, got", actual)
	}

	expected := errors.New("foo")

	r = CallbackReadCloser(nil, func() error {
		return expected
	})

	if actual := r.Close(); actual != expected {
		t.Error("Expected:", expected, "got:", actual)
	}

	buf := []byte("somebytes")
	r = CallbackReadCloser(bytes.NewBuffer(buf), nil)
	content, err := ioutil.ReadAll(r)
	if err != nil {
		t.Errorf("Expected a non nil error, got: %s", err)
	}
	if string(buf) != string(content) {
		t.Errorf("Expected == %s, Actual == %s", buf, content)
	}
}
