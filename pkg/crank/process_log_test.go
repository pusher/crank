package crank

import (
	"math/rand"
	"strings"
	"testing"
)

func TestLinePrefixer(t *testing.T) {
	xxx := func() string { return "XXX" }
	cmp(t, "hello\nworld", "XXXhello\nXXXworld", xxx)
	cmp(t, "hello\nworld\n", "XXXhello\nXXXworld\n", xxx)

	randCmp(t, "this\nis\n\nmy\ninput text", "XXXthis\nXXXis\nXXX\nXXXmy\nXXXinput text", xxx)

}

func cmp(t *testing.T, input, output string, prefix func() string) {
	r := setup(input, prefix)

	buf := make([]byte, 1024)

	n, err := r.Read(buf)
	if err != nil {
		t.Error(err)
	}

	if string(buf[:n]) != output {
		t.Errorf("%v != %v", buf[:n], []byte(output))
	}

	n, err = r.Read(buf)
	if n > 0 {
		t.Error(">0 ", n)
	}
	if err.Error() != "EOF" {
		t.Error("foo", err)
	}
}

func randCmp(t *testing.T, input, output string, prefix func() string) {
	out := ""
	r := setup(input, prefix)
	for {
		buf := make([]byte, int(rand.Float32()*13))
		n, err := r.Read(buf)
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			t.Error(err)
		}
		out += string(buf[:n])
	}

	if out != output {
		t.Errorf("%v != %v", out, output)
	}
}

func setup(input string, prefix func() string) *PrefixReader {
	io := strings.NewReader(input)
	return NewLinePrefixer(io, prefix)
}
