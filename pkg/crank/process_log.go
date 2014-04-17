package crank

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"time"
)

type processLog struct {
	out io.Writer
	p   *Process
}

func newProcessLog(out io.Writer, p *Process) *processLog {
	return &processLog{out, p}
}

func (self *processLog) copy(r io.Reader) {
	// Use scanner to read lines from the input
	scanner := bufio.NewScanner(r)
	var line string

	for scanner.Scan() {
		// e.g. Mar 18 10:08:13.839 (1)[69282] Logentry
		line = fmt.Sprintln(
			time.Now().Format(time.StampMilli),
			fmt.Sprintf("[%v]", self.p.Pid),
			scanner.Text(),
		)
		self.out.Write([]byte(line))
	}

	if err := scanner.Err(); err != nil {
		log.Println("ERROR:", err)
	}
}
