package main

import (
	"encoding/json"
	"errors"
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
	*EventLoop
	proto        *Prototype
	exitHandlers []ProcessExitCallback
	_sendSignal  chan syscall.Signal
	notify       *os.File
	command      *exec.Cmd
	pid          int
	onStarted    chan bool
}

type ProcessExitCallback func(p *Process)

func NewProcess(proto *Prototype, started chan bool) *Process {
	return &Process{
		EventLoop:    NewEventLoop(),
		proto:        proto,
		exitHandlers: make([]ProcessExitCallback, 0),
		_sendSignal:  make(chan syscall.Signal),
		onStarted:    started,
	}
}

func (p *Process) String() string {
	return fmt.Sprintf("[%v] ", p.pid)
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

	command := exec.Command(p.proto.cmd, p.proto.args...)
	p.command = command

	// Inherit the environment with which crank was run
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")
	command.Env = append(command.Env, "NOTIFY_FD=4")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.proto.fd) // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, notifySnd)  // 4: notify socket

	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()
	command.Stdin = DevNull

	// Start process
	if err = command.Start(); err != nil {
		log.Fatal("Process start failed: ", err)
	}
	p.pid = command.Process.Pid
	p.Log("Process started")

	// Write stdout & stderr to the
	processLog := NewProcessLog(p.proto.out, p.pid)
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

		p.NextTick(func() {
			for _, f := range p.exitHandlers {
				f(p)
			}
			p.EventLoop.Stop()
		})
	}()

	// Main run loop for process
	go p.EventLoop.Run(time.Second, NoopCallback)
}

// Shutdown send a SIGTERM signal to the process and lets the process gracefully
// shutdown.
func (p *Process) Shutdown() {
	p.NextTick(func() {
		p.sendSignal(syscall.SIGTERM)
	})
}

// Stop stops the process with increased aggressiveness
func (p *Process) Stop() {
	p.NextTick(func() {
		p.sendSignal(syscall.SIGTERM)
	})
	p.AddTimer(1*time.Second, func() {
		p.sendSignal(syscall.SIGTERM)
	})
	p.AddTimer(2*time.Second, func() {
		p.sendSignal(syscall.SIGKILL)
	})
}

// Register a function to be called when the process exists
func (p *Process) OnExit(f ProcessExitCallback) {
	p.exitHandlers = append(p.exitHandlers, f)
}

func (p *Process) sendSignal(sig syscall.Signal) {
	p.Log("Sending signal: %v", sig)
	p.command.Process.Signal(sig)
}

func decodePipeCommand(data []byte) (err error, command string, args interface{}) {
	var obj interface{}

	if err = json.Unmarshal(data, &obj); err != nil {
		return
	} else {
		switch arr := obj.(type) {
		default:
			err = errors.New("Invalid protocol")
			return
		case []interface{}:
			args = arr[1]

			switch c := arr[0].(type) {
			case string:
				command = c
				return
			default:
				return
			}
		}
	}
}
