// Package testio implements various io utility types. Included are
// BrokenWriter, which fails after writing a certain number of bytes;
// a BufCloser, which wraps a bytes.Buffer in a Close method; and a
// LoggingBuffer that logs all reads and writes.
package testio

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
)

// BrokenWriter implements an io.Writer that fails after a certain
// number of bytes. This can be used to simulate a network connection
// that breaks during write or a file on a filesystem that becomes
// full, for example. A BrokenWriter doesn't actually store any data.
type BrokenWriter struct {
	current, limit int
}

// NewBrokenWriter creates a new BrokenWriter that can store only
// limit bytes.
func NewBrokenWriter(limit int) *BrokenWriter {
	return &BrokenWriter{limit: limit}
}

// Write will write the byte slice to the BrokenWriter, failing if the
// maximum number of bytes has been reached.
func (w *BrokenWriter) Write(p []byte) (int, error) {
	if (len(p) + w.current) <= w.limit {
		w.current += len(p)
	} else {
		spill := (len(p) + w.current) - w.limit
		w.current = w.limit
		return len(p) - spill, errors.New("write failed")
	}

	return len(p), nil
}

// Extend increases the byte limit to allow more data to be written.
func (w *BrokenWriter) Extend(n int) {
	w.limit += n
}

// Reset clears the limit and bytes in the BrokenWriter. Extend needs
// to be called to allow data to be written.
func (w *BrokenWriter) Reset() {
	w.limit = 0
	w.current = 0
}

// BrokenReadWriter implements a broken reader and writer, backed by a
// bytes.Buffer.
type BrokenReadWriter struct {
	limit, current int
	buf            *bytes.Buffer
}

// NewBrokenReadWriter initialises a new BrokerReadWriter with an empty
// reader and the specified limit.
func NewBrokenReadWriter(limit int) *BrokenReadWriter {
	return &BrokenReadWriter{
		limit: limit,
		buf:   &bytes.Buffer{},
	}
}

// Write satisfies the Writer interface.
func (brw *BrokenReadWriter) Write(p []byte) (int, error) {
	if (len(p) + brw.buf.Len()) > brw.limit {
		remain := brw.limit - brw.buf.Len()
		if remain > 0 {
			brw.buf.Write(p[:remain])
			return remain, errors.New("testio: write failed")
		}
		return 0, errors.New("testio: write failed")
	}
	return brw.buf.Write(p)
}

// Read satisfies the Reader interface.
func (brw *BrokenReadWriter) Read(p []byte) (int, error) {
	n, err := brw.buf.Read(p)
	brw.current -= n
	return n, err
}

// Extend increases the BrokenReadWriter limit.
func (brw *BrokenReadWriter) Extend(n int) {
	brw.limit += n
}

// Reset clears the internal buffer. It retains its original limit.
func (brw *BrokenReadWriter) Reset() {
	brw.buf.Reset()
}

// BufCloser is a buffer wrapped with a Close method.
type BufCloser struct {
	buf *bytes.Buffer
}

// Write writes the data to the BufCloser.
func (buf *BufCloser) Write(p []byte) (int, error) {
	return buf.buf.Write(p)
}

// Read reads data from the BufCloser.
func (buf *BufCloser) Read(p []byte) (int, error) {
	return buf.buf.Read(p)
}

// Close is a stub function to satisfy the io.Closer interface.
func (buf *BufCloser) Close() {}

// Reset clears the internal buffer.
func (buf *BufCloser) Reset() {
	buf.buf.Reset()
}

// Bytes returns the contents of the buffer as a byte slice.
func (buf *BufCloser) Bytes() []byte {
	return buf.buf.Bytes()
}

// NewBufCloser creates and initializes a new BufCloser using buf as
// its initial contents. It is intended to prepare a BufCloser to read
// existing data. It can also be used to size the internal buffer for
// writing. To do that, buf should have the desired capacity but a
// length of zero.
func NewBufCloser(buf []byte) *BufCloser {
	bc := new(BufCloser)
	bc.buf = bytes.NewBuffer(buf)
	return bc
}

// NewBufCloserString creates and initializes a new Buffer using
// string s as its initial contents. It is intended to prepare a
// buffer to read an existing string.
func NewBufCloserString(s string) *BufCloser {
	buf := new(BufCloser)
	buf.buf = bytes.NewBufferString(s)
	return buf
}

// A LoggingBuffer is an io.ReadWriter that prints the hex value of
// the data for all reads and writes.
type LoggingBuffer struct {
	rw   io.ReadWriter
	w    io.Writer
	name string
}

// NewLoggingBuffer creates a logging buffer from an existing
// io.ReadWriter. By default, it will log to standard error.
func NewLoggingBuffer(rw io.ReadWriter) *LoggingBuffer {
	return &LoggingBuffer{
		rw: rw,
		w:  os.Stderr,
	}
}

// LogTo sets the io.Writer that the buffer will write logs to.
func (lb *LoggingBuffer) LogTo(w io.Writer) {
	lb.w = w
}

// SetName gives a name to the logging buffer to help distinguish
// output from this buffer.
func (lb *LoggingBuffer) SetName(name string) {
	lb.name = name
}

// Write writes the data to the logging buffer and writes the data to
// the logging writer.
func (lb *LoggingBuffer) Write(p []byte) (int, error) {
	if lb.name != "" {
		fmt.Fprintf(lb.w, "[%s] ", lb.name)
	}

	fmt.Fprintf(lb.w, "[WRITE] %x\n", p)
	return lb.rw.Write(p)
}

// Read reads the data from the logging buffer and writes the data to
// the logging writer.
func (lb *LoggingBuffer) Read(p []byte) (int, error) {
	n, err := lb.rw.Read(p)
	if err != nil {
		return n, err
	}
	if lb.name != "" {
		fmt.Fprintf(lb.w, "[%s] ", lb.name)
	}

	fmt.Fprintf(lb.w, "[READ] %x\n", p)
	return n, err
}
