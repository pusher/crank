package crank

import (
	"bytes"
	"io"
	"os"
)

var EMPTY_BYTES = []byte{}

func startProcessLogger(out io.Writer, prefix func() string) (w *os.File, err error) {
	var r *os.File

	r, w, err = os.Pipe()
	if err != nil {
		return
	}

	go runProcesssLogger(out, r, prefix)

	return w, nil
}

func runProcesssLogger(out io.Writer, r *os.File, prefix func() string) {
	defer r.Close()

	_, err := io.Copy(out, NewLinePrefixer(r, prefix))

	if err != nil {
		fail(err)
	}
}

type PrefixReader struct {
	r          io.Reader
	prefix     func() string
	buf        []byte
	needPrefix bool
}

func NewLinePrefixer(r io.Reader, prefix func() string) *PrefixReader {
	return &PrefixReader{r, prefix, EMPTY_BYTES, true}
}

func (io *PrefixReader) Read(buf []byte) (n int, err error) {

	if len(io.buf) == 0 { // Get more data
		var nr, i int

		// If len(buf) is fixed we make regular reads
		buf2 := make([]byte, len(buf))
		nr, err = io.r.Read(buf2)
		if err != nil {
			return
		}

		buf2 = buf2[:nr]

		// Prefix and put buf2 into io.buf
		for len(buf2) > 0 {
			if io.needPrefix {
				io.buf = append(io.buf, []byte(io.prefix())...)
				io.needPrefix = false
			}

			i = bytes.IndexRune(buf2, '\n')
			if i >= 0 {
				io.buf = append(io.buf, buf2[:i+1]...)
				buf2 = buf2[i+1:]
				io.needPrefix = true
			} else {
				io.buf = append(io.buf, buf2...)
				buf2 = EMPTY_BYTES
			}
		}
	}

	// Get io.buf data into buf
	n = minInt(len(buf), len(io.buf))
	for i, b := range io.buf[:n] {
		buf[i] = b
	}
	io.buf = io.buf[n:]
	return
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}
