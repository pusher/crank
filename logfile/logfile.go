package logfile

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

type Logfile struct {
	filename  string
	sigUSR1   chan os.Signal
	outFile   *os.File
	outWriter *bufio.Writer
	_write    chan []byte
	_close    chan bool
	_didClose chan bool
}

func New(filename string) *Logfile {
	sigUSR1 := make(chan os.Signal, 1)
	signal.Notify(sigUSR1, syscall.SIGUSR1)

	l := &Logfile{
		filename:  filename,
		sigUSR1:   sigUSR1,
		_write:    make(chan []byte),
		_close:    make(chan bool),
		_didClose: make(chan bool),
	}
	l.createOutput()
	return l
}

// Write conforms to Writer in interface, but maybe not in spirit
func (l *Logfile) Write(data []byte) (n int, err error) {
	l._write <- data
	return len(data), nil
}

func (l *Logfile) WriteLine(data []byte) {
	data = append(data, []byte("\r\n")...)
	l._write <- data
}

func (l *Logfile) WriteString(data string) {
	l._write <- []byte(data + "\r\n")
}

func (l *Logfile) Println(v ...interface{}) {
	l._write <- []byte(fmt.Sprintln(v...))
}

func (l *Logfile) Close() {
	l._close <- true
	<-l._didClose
}

func (l *Logfile) Run() {
	for {
		select {
		case data := <-l._write:
			l.write(data)
		case <-l._close:
			l.close()
			l._didClose <- true
		case <-l.sigUSR1:
			l.close()
			l.createOutput()
		}
	}
}

func (l *Logfile) write(data []byte) {
	if l.outWriter == nil {
		return
	}
	if _, err := l.outWriter.Write(data); err != nil {
		log.Printf("Error writing to file: %v", err)
		return
	}
}

func (l *Logfile) close() {
	if l.outFile != nil {
		l.outWriter.Flush()
		l.outFile.Close()
	}
}

func (l *Logfile) createOutput() {
	log.Printf("Writing new output file: %v", l.filename)
	if f, err := os.Create(l.filename); err != nil {
		log.Printf("Error creating output file %v: %v", l.filename, err)
	} else {
		l.outFile = f
		l.outWriter = bufio.NewWriter(f)
	}
}
