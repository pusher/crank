package crank

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"time"
)

// FIXME: the logger will shutdown if a line is bigger than bufio.MaxScanTokenSize
//        (64k at the moment)

func startProcessLogger(out io.Writer, tag func() string) (w *os.File, err error) {
	var r *os.File

	r, w, err = os.Pipe()
	if err != nil {
		return
	}

	go runProcesssLogger(out, r, tag)

	return w, nil
}

func runProcesssLogger(out io.Writer, r *os.File, tag func() string) {
	defer r.Close()

	var line string

	// Use scanner to read lines from the input
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		// e.g. Mar 18 10:08:13.839 (1)[69282] Logentry
		line = fmt.Sprintln(
			time.Now().Format(time.StampMilli),
			tag(),
			scanner.Text(),
		)
		out.Write([]byte(line))
	}

	if err := scanner.Err(); err != nil {
		fail(err)
	}
}
