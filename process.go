package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

var DevNull *os.File

func init() {
	var err error
	if DevNull, err = os.Open("/dev/null"); err != nil {
		panic("could not open /dev/null: " + err.Error())
	}
}

type Process struct {
	*os.Process
	config      *ProcessConfig
	external    *External
	_sendSignal chan syscall.Signal
	notify      *os.File
	onStarted   chan bool
	onExited    chan *Process
}

func NewProcess(config *ProcessConfig, external *External, started chan bool, exited chan *Process) *Process {
	return &Process{
		config:      config,
		external:    external,
		_sendSignal: make(chan syscall.Signal),
		onStarted:   started,
		onExited:    exited,
	}
}

func (p *Process) String() string {
	return fmt.Sprintf("[%v] ", p.Pid)
}

func (p *Process) Log(format string, v ...interface{}) {
	log.Print(p.String(), fmt.Sprintf(format, v...))
}

func (p *Process) Start() {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		log.Fatal("Process start failed: ", err)
	}
	notifyRcv := os.NewFile(uintptr(fds[0]), "<-|->") // File name is arbitrary
	notifySnd := os.NewFile(uintptr(fds[1]), "--({_O_})--")

	command := exec.Command(p.config.Command)

	// Inherit the environment with which crank was run
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")
	command.Env = append(command.Env, "NOTIFY_FD=4")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.external.Fd) // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, notifySnd)     // 4: notify socket

	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()
	command.Stdin = DevNull

	// Start process
	if err = command.Start(); err != nil {
		log.Fatal("Process start failed: ", err)
	}
	p.Process = command.Process
	p.Log("Process started")

	// Write stdout & stderr to the
	processLog := NewProcessLog(os.Stdout, p.Pid)
	go processLog.Copy(stdout)
	go processLog.Copy(stderr)

	// Close unused pipe-ends
	notifySnd.Close()

	// Read on pipe from child, and process commands
	go func() {
		defer notifyRcv.Close()

		var err error
		var command string
		var n int
		data := make([]byte, 4096)

		for {
			n, err = notifyRcv.Read(data)
			if err != nil {
				p.Log("Error reading on pipe: %v", err)
				return
			}

			command = strings.TrimSpace(string(data[:n]))

			p.Log("Received command on pipe: %v", command)

			switch command {
			case "READY=1":
				p.onStarted <- true
				p.Log("After onStarted")
			default:
				p.Log("Unknown command received: %v", command)
			}
		}
	}()

	// Goroutine catches process exit
	go func() {
		if err := command.Wait(); err == nil {
			p.Log("Exited cleanly")
		} else {
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					// Prints the cause of exit - either exit status or signal should be
					// != -1 (-1 means not exited or not signaled). See
					// http://golang.org/pkg/syscall/#WaitStatus
					p.Log(
						"Unclean exit: %v (exit status: %v, signal: %v)",
						err, status.ExitStatus(), int(status.Signal()),
					)
				} else {
					p.Log("Unsupported ExitError: %v", err)
				}
			} else {
				p.Log("Unexpected exit: %v", err)
			}
		}

		p.onExited <- p
	}()
}

// Shutdown send a SIGTERM signal to the process and lets the process gracefully
// shutdown.
func (p *Process) Shutdown() {
	p.sendSignal(syscall.SIGTERM)
}

func (p *Process) Kill() {
	p.sendSignal(syscall.SIGKILL)
}

// Stop stops the process with increased aggressiveness
func (p *Process) Stop() {
	p.sendSignal(syscall.SIGTERM)
	// TODO do we need to stop this timer if the process exits earlier?
	time.AfterFunc(2*time.Second, func() {
		p.sendSignal(syscall.SIGKILL)
	})
}

func (p *Process) sendSignal(sig syscall.Signal) {
	p.Log("Sending signal: %v", sig)
	p.Signal(sig)
}
