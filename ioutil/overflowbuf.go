// package ioutil provides additional io related helpers beyond what can be found in the go standard library
package ioutil

import (
	"os"
	"sync"
	"io/ioutil"
	"fmt"
	"errors"
	"io"
)

var freeOverflowBuffers = sync.Pool{
	New: func() interface{} { return new(OverflowBuffer) },
}

// OverflowBuffer is a byte buffer that overflows to disk when its capacity has been reached
type OverflowBuffer struct {
	// Capacity is the number of bytes that the buffer will hold before writing to disk
	Capacity      int
	// Dir and Prefix control the location of the backing files in the same manner as the standard library's ioutil.TempFile
	Dir, Prefix   string

	buf           []byte
	nwrote, nread int
	fileReset     bool
	f             *os.File
}

// GetOverflowBufferFromPool returns an OverflowBuffer with the given Capacity, Dir and Prefix from a sync.Pool
func GetOverflowBufferFromPool(capacity int, dir, prefix string) *OverflowBuffer {
	fb := freeOverflowBuffers.Get().(*OverflowBuffer)
	fb.Capacity = capacity
	fb.Dir = dir
	fb.Prefix = prefix
	return fb
}

// ReleaseOverflowBufferToPool will return small capacity buffers to a sync.Pool
func ReleaseOverflowBufferToPool(ob *OverflowBuffer) {
	if cap(ob.buf) > 2048 {
		return
	}
	ob.buf = ob.buf[:0]
	ob.nwrote = 0
	ob.nread = 0
	ob.f = nil
	ob.fileReset = false
	freeOverflowBuffers.Put(ob)
}

// Read implements io.Reader. After calling Read, any call to Write will return an error.
func (ob *OverflowBuffer) Read(p []byte) (nread int, err error) {
	defer func() {
		if err != nil && err != io.EOF {
			err = fmt.Errorf("OverflowBuffer.Read: %s", err)
		}
	}()

	nread = copy(p, ob.buf[ob.nread:])
	ob.nread += nread

	if len(p) > nread {
		if ob.f == nil {
			err = io.EOF
			return
		}
		if !ob.fileReset {
			ob.fileReset = true
			_, err = ob.f.Seek(0, 0)
			if err != nil {
				return
			}
		}
		var n int
		n, err = ob.f.Read(p[nread:])
		nread += n
	}

	return
}

// Write implements io.Writer. If called after Read of ResetRead, an error will be returned
func (ob *OverflowBuffer) Write(p []byte) (nwrote int, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("OverflowBuffer.Write: %s", err)
		}
	}()

	if ob.nread > 0 {
		err = errors.New("write called after Read or ResetRead")
		return
	}

	nwrote = ob.Capacity - ob.nwrote
	if nwrote > len(p) {
		nwrote = len(p)
	}

	ob.buf = append(ob.buf, p[:nwrote]...)
	ob.nwrote += nwrote

	if len(p) > nwrote {
		if ob.f == nil {
			var f *os.File
			f, err = ioutil.TempFile(ob.Dir, ob.Prefix)
			if err != nil {
				return
			}
			ob.f = f
		}
		var n int
		n, err = ob.f.Write(p[nwrote:])
		nwrote += n
	}

	return
}

// Close implements io.Closer. Calling Close will remove any backing file that was created as a result of overflowing the capacity.
func (ob *OverflowBuffer) Close() (err error) {
	if ob.f != nil {
		ob.f.Close()
		err = os.Remove(ob.f.Name())
	}
	return
}
