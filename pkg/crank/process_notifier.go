package crank

import (
	"io"
	"log"
	"os"
	"strings"
)

type processNotifier struct {
	file  *os.File
	ready chan<- bool
}

func newProcessNotifier(file *os.File, ready chan<- bool) *processNotifier {
	return &processNotifier{file, ready}
}

func (self *processNotifier) run() {
	// Read on pipe from child, and process commands
	defer self.file.Close()

	var err error
	var command string
	var n int
	data := make([]byte, 4096)

	for {
		n, err = self.file.Read(data)
		if err == io.EOF {
			return
		}
		if err != nil {
			log.Printf("Error reading on pipe: %v", err)
			return
		}

		command = strings.TrimSpace(string(data[:n]))

		switch command {
		case "READY=1":
			self.ready <- true
		default:
			log.Println("Unknown command received: ", command)
		}
	}
}
