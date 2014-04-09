package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"time"
)

type ProcessLog struct {
	out    io.Writer
	prefix string
}

func NewProcessLog(out io.Writer, pid int) *ProcessLog {
	prefix := fmt.Sprintf("[%v]", pid)
	return &ProcessLog{out, prefix}
}

func (self *ProcessLog) Copy(r io.Reader) {
	// Use scanner to read lines from the input
	scanner := bufio.NewScanner(r)
	var line string

	for scanner.Scan() {
		// e.g. Mar 18 10:08:13.839 (1)[69282] Logentry
		line = fmt.Sprintln(
			time.Now().Format(time.StampMilli),
			self.prefix,
			scanner.Text(),
		)
		self.out.Write([]byte(line))
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
