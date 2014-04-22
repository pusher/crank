package crank

import (
	"io"
	"log"
	"os"
	"strings"
	"syscall"
)

// TODO: The notifier can potentially have other notifications than "ready".
//       Eg: heartbeat

// Gets a channel on which to publish events.
//
// Returns a file on which the process is supposed to write data, which then
// translate into these events.
func startProcessNotifier(ready chan<- bool) (w *os.File, err error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return
	}
	r := os.NewFile(uintptr(fds[0]), "notify:r") // File name is arbitrary
	w = os.NewFile(uintptr(fds[1]), "notify:w")

	go runProcessNotifier(r, ready)

	return w, nil
}

func runProcessNotifier(r *os.File, ready chan<- bool) {
	// Read on pipe from child, and process commands
	defer r.Close()
	defer close(ready)

	var err error
	var command string
	var n int
	data := make([]byte, 4096)

	for {
		n, err = r.Read(data)
		if err == io.EOF {
			return
		}
		if err != nil {
			fail("Reading on pipe", err)
			return
		}

		command = strings.TrimSpace(string(data[:n]))

		switch command {
		case "READY=1":
			ready <- true
		default:
			log.Println("Unknown command received: ", command)
		}
	}
}
