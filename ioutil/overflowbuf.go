// package ioutil provides additional io related helpers beyond what can be found in the go standard library
package ioutil

import (
	"os"
	"sync"
	"io/ioutil"
	"fmt"
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
	fWrite, fRead *os.File
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
	ob.fWrite = nil
	ob.fRead = nil
	freeOverflowBuffers.Put(ob)
}

// Read implements io.Reader
func (ob *OverflowBuffer) Read(p []byte) (nread int, err error) {
	defer func() {
		if err != nil && err != io.EOF {
			err = fmt.Errorf("OverflowBuffer.Read: %s", err)
		}
	}()

	nread = copy(p, ob.buf[ob.nread:])
	ob.nread += nread

	if len(p) > nread {
		if ob.fWrite == nil {
			err = io.EOF
			return
		}
		if ob.fRead == nil {
			ob.fRead, err = os.Open(ob.fWrite.Name())
			if err != nil {
				return
			}
		}
		var n int
		n, err = ob.fRead.Read(p[nread:])
		nread += n
	}

	return
}

// Write implements io.Writer
func (ob *OverflowBuffer) Write(p []byte) (nwrote int, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("OverflowBuffer.Write: %s", err)
		}
	}()

	nwrote = ob.Capacity - ob.nwrote
	if nwrote > len(p) {
		nwrote = len(p)
	}

	ob.buf = append(ob.buf, p[:nwrote]...)
	ob.nwrote += nwrote

	if len(p) > nwrote {
		if ob.fWrite == nil {
			ob.fWrite, err = ioutil.TempFile(ob.Dir, ob.Prefix)
			if err != nil {
				return
			}
		}
		var n int
		n, err = ob.fWrite.Write(p[nwrote:])
		nwrote += n
	}

	return
}

// Close implements io.Closer. Calling Close will remove any backing file that was created as a result of overflowing the capacity.
func (ob *OverflowBuffer) Close() (err error) {
	if ob.fRead != nil {
		ob.fRead.Close()
	}
	if ob.fWrite != nil {
		ob.fWrite.Close()
		err = os.Remove(ob.fWrite.Name())
	}
	return
}

// ResetRead will set the buffer to read from the beginning.
func (ob *OverflowBuffer) ResetRead() error {
	ob.nread = 0
	if ob.fRead != nil {
		_, err := ob.fRead.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("OverflowBuffer.ResetRead: %s", err)
		}
	}
	return nil
}