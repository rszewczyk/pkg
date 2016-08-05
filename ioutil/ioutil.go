package ioutil

import "io"

type readCloser struct {
	io.Reader
	closeCallback func() error
}

func (rc *readCloser) Close() error {
	if rc.closeCallback == nil {
		return nil
	}

	return rc.closeCallback()
}

// CallbackReadCloser wraps the given reader in an io.ReadCloser. The onClose callback will be called (if non-nil)
// when the wrapping ReadCloser's Close method is called
func CallbackReadCloser(r io.Reader, onClose func() error) io.ReadCloser {
	return &readCloser{r, onClose}
}
