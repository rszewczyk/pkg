package ioutil

import (
	"testing"
	"io/ioutil"
	"bytes"
	"os"
	"math/rand"
	"fmt"
)

func check(t *testing.T, testName string, ob *OverflowBuffer, pp []byte) {
	content, err := ioutil.ReadAll(ob)
	if err != nil {
		t.Errorf(testName+" check(1): err should be nil, found err == %s", err)
	}
	if a, b := len(content), len(pp); a != b {
		t.Errorf(testName+" check(2): len(content) == %d, len(buf) == %d", a, b)
	}
	if bytes.Compare(content, pp) != 0 {
		t.Errorf(testName+" check(3): content == %s, buf == %s", string(content), string(pp))
	}
}

func fill(t *testing.T, testName string, ob *OverflowBuffer, p []byte, n int) (result []byte) {
	for ; n > 0; n-- {
		m, err := ob.Write(p)
		if err != nil {
			t.Errorf(testName+" fill(1): err should be nil, found err == %s", err)
		}
		if e := len(p); m != e {
			t.Errorf(testName+" fill(2): m == %d, expected %d", m, e)
		}
		result = append(result, p...)
	}

	return result
}

func cleanup(t *testing.T, testName string, ob *OverflowBuffer) {
	var fnWrite string
	if ob.fWrite != nil {
		fnWrite = ob.fWrite.Name()
	}
	err := ob.Close()
	if err != nil {
		t.Errorf(testName+" close (1): err should be nil, found err == %s", err)
	}

	_, err = os.Stat(fnWrite)
	if _, ok := err.(*os.PathError); !ok {
		t.Errorf(testName+" close (2): file should have been deleted (err should be non nil and of type *os.PathError), found err == %s", err)
	}
}

func TestGetOverflowBufferFromPool(t *testing.T) {
	tests := []struct{
		capacity int
		prefix, dir string
	}{
		{13, "foo", "/bar"},
		{128, "aabbcc", "/a/b/c"},
		{0, "", ""},
	}

	for i, test := range tests {
		testName := fmt.Sprintf("TestNewOverflowBuffer loop (%d)", i)
		ob := GetOverflowBufferFromPool(test.capacity, test.dir, test.prefix)
		if ob.Capacity != test.capacity {
			t.Errorf(testName+" (%d) (1): ob.Capacity == %d, expected == %d", i, ob.Capacity, test.capacity)
		}
		if ob.Dir != test.dir {
			t.Errorf(testName+" (%d) (2): ob.Dir == %s, expected == %s", i, ob.Dir, test.dir)
		}
		if ob.Prefix != test.prefix {
			t.Errorf(testName+" (%d) (3): ob.Prefix == %s, expected %s", i, ob.Prefix, test.prefix)
		}
		cleanup(t, testName, ob)
	}
}

func TestReleaseOverflowBufferToPool(t *testing.T) {
	testName := "TestReleaseOverflowBufferToPool"
	fWrite, _ := ioutil.TempFile("", "")
	fRead, _ := os.Open(fWrite.Name())

	ob := &OverflowBuffer{
		buf: make([]byte, 100, 100),
		nwrote: 100,
		nread: 100,
		fWrite: fWrite,
		fRead: fRead,
		eof: true,
	}

	ob.Close()

	ReleaseOverflowBufferToPool(ob)

	if len(ob.buf) > 0 {
		t.Errorf(testName+" (1): Expected length of internal buffer to be zero, got %d", len(ob.buf))
	}
	if ob.nread != 0 {
		t.Errorf(testName+" (2): Expected nread to be zero, got %d", ob.nread)
	}
	if ob.nwrote != 0 {
		t.Errorf(testName+" (3): Expected nwrote to be zero, got %d", ob.nwrote)
	}
	if ob.fWrite != nil {
		t.Errorf(testName+" (4): Expected f to be nil, got file with name %s", ob.fWrite.Name())
	}
	if ob.fRead != nil {
		t.Errorf(testName+" (5): Expected f to be nil, got file with name %s", ob.fRead.Name())
	}
	if ob.eof {
		t.Error(testName+" (6): Expected eof to be false")
	}
}

func TestOverflowBufferBasicOperations(t *testing.T) {
	testName := "TestOverflowBufferBasicOperations"
	ob := &OverflowBuffer{ Capacity: 4 }

	p1, p2 := []byte("abcd"), []byte("efgh")
	n, err := ob.Write(p1)
	if e := len(p1); n != e {
		t.Errorf(testName+" (1): n == %d, expected %d", n, e)
	}
	if err != nil {
		t.Errorf(testName+" (2): err should be nil, found err == %s", err)
	}

	if ob.fWrite != nil {
		t.Errorf(testName+" (3): expected ob.f to be nil, found a file with name == %s", ob.fWrite.Name())
	}

	n, err = ob.Write(p2)
	if e := len(p2); n != e {
		t.Errorf(testName+" (4): n == %d, expected %d", n, e)
	}
	if err != nil {
		t.Errorf(testName+" (5): err should be nil, found err == %s", err)
	}

	if ob.fWrite == nil {
		t.Errorf(testName+" (6): expected ob.f to be non nil")
	}

	check(t, testName+" (7)", ob, append(p1, p2...))

	ob.ResetRead()

	check(t, testName+" (8)", ob, append(p1, p2...))

	cleanup(t, testName, ob)
}

func TestOverflowBufferWriteCrossesFileBoundary(t *testing.T) {
	testName := "TestWriteCrossesFileBoundary"
	ob := &OverflowBuffer{ Capacity: 10 }

	p1, p2 := []byte("abcdef"), []byte("ghijklmn")
	n, err := ob.Write(p1)
	if e := len(p1); n != e {
		t.Errorf(testName+" (1): n == %d, expected %d", n, e)
	}
	if err != nil {
		t.Errorf(testName+" (2): err should be nil, found err == %s", err)
	}

	if ob.fWrite != nil {
		t.Errorf(testName+" (3): expected ob.f to be nil, found a file with name == %s", ob.fWrite.Name())
	}

	n, err = ob.Write(p2)
	if e := len(p2); n != e {
		t.Errorf(testName+" (4): n == %d, expected %d", n, e)
	}
	if err != nil {
		t.Errorf(testName+" (5): err should be nil, found err == %s", err)
	}

	if ob.fWrite == nil {
		t.Errorf(testName+" (6): expected ob.f to be non nil")
	}

	check(t, testName+" (7)", ob, append(p1, p2...))
	cleanup(t, testName, ob)
}

func TestLongWriteLongBuffer(t *testing.T) {
	testName := "TestLongWrite"
	ob := &OverflowBuffer{ Capacity: 4000 }
	r1 := fill(t, testName, ob, []byte("abcdefgh"), 500)
	r2 := fill(t, testName, ob, []byte("ijklmnop"), 500)
	check(t, testName, ob, append(r1, r2...))
}

func TestLongWriteShortBuffer(t *testing.T) {
	testName := "TestShortBuffer"
	ob := &OverflowBuffer{ Capacity: 10 }
	r1 := fill(t, testName, ob, []byte("abcdefgh"), 500)
	r2 := fill(t, testName, ob, []byte("ijklmnop"), 500)
	check(t, testName, ob, append(r1, r2...))
}

func TestRandomWrites(t *testing.T) {
	for i := 0; i < 100; i++ {
		var p []byte
		capacity := rand.Intn(1000) + 1
		ob := &OverflowBuffer{ Capacity: capacity }
		for i := 0; i < rand.Intn(10) + 1; i++ {
			p = append(p, 'a' + byte(rand.Intn(26)))
		}
		n := capacity
		testName := fmt.Sprintf("TestRandomWrites loop (%d), capacity == %d, p == %s, n == %d", i, capacity, string(p), n)
		r := fill(t, testName, ob, p, n)
		check(t, testName, ob, r)
		ob.ResetRead()
		check(t, testName, ob, r)
		cleanup(t, testName, ob)
	}
}

func TestZeroLengthRead(t *testing.T) {
	buffers := []*OverflowBuffer{
		{ Capacity: 0 },
		{ Capacity: 100 },
	}

	for _, buf := range buffers {
		_, err := ioutil.ReadAll(buf)
		if err != nil {
			t.Errorf("TestZeroLengthRead: expected err to be nil, got %s", err)
		}
	}
}
