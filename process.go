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

const (
	PROCESS_NEW = ProcessState(iota)
	PROCESS_STARTING
	PROCESS_RUNNING
	PROCESS_STOPPING
	PROCESS_STOPPED
)

type ProcessState int

type ExitStatus struct {
	code int
	err  error
}

func init() {
	var err error
	if DevNull, err = os.Open("/dev/null"); err != nil {
		panic("could not open /dev/null: " + err.Error())
	}
}

type Process struct {
	*os.Process
	state       ProcessState
	config      *ProcessConfig
	external    *External
	_sendSignal chan syscall.Signal
	notify      *os.File
	onStarted   chan bool
	onExited    chan *Process
}

func NewProcess(config *ProcessConfig, external *External, started chan bool, exited chan *Process) *Process {
	return &Process{
		state:       PROCESS_NEW,
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
		p.state = PROCESS_STOPPED
		log.Fatal("Process start failed: ", err)
	}
	p.state = PROCESS_STARTING
	p.Process = command.Process
	p.Log("Process started")

	// Write stdout & stderr to the
	processLog := NewProcessLog(os.Stdout, p.Pid)
	go processLog.Copy(stdout)
	go processLog.Copy(stderr)

	// Close unused pipe-ends
	notifySnd.Close()

	started := make(chan bool)
	exited := make(chan *ExitStatus)

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
				started <- true
			default:
				p.Log("Unknown command received: %v", command)
			}
		}
	}()

	// Goroutine catches process exit
	go func() {
		err := command.Wait()
		exited <- getExitStatusCode(err)
	}()

	go func() {
		for {
			switch p.state {
			case PROCESS_STARTING:
				select {
				case <-time.After(time.Duration(p.config.StartTimeout) * time.Millisecond):
					p.Log("Process did not start in time, killing")
					p.Kill()
				case <-started:
					p.Log("Process transitioning to running")
					p.state = PROCESS_RUNNING
					p.onStarted <- true
				case <-exited:
					p.Log("Process exited while starting")
					p.state = PROCESS_STOPPED
				}

			case PROCESS_RUNNING:
				select {
				case <-started:
					p.Log("Process started twice")
				case <-exited:
					p.Log("Process exited while running")
					p.state = PROCESS_STOPPED
				}

			case PROCESS_STOPPING:
				select {
				case <-time.After(time.Duration(p.config.StopTimeout) * time.Millisecond):
					p.Log("Process did not stop in time, killing")
					p.Kill()
				case <-exited:
					p.state = PROCESS_STOPPED
				}

			case PROCESS_STOPPED:
				p.Log("Process stopped")
				p.onExited <- p
				return

			default:
				panic(fmt.Sprintf("BUG, unknown state %v", p.state))
			}
		}
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

func getExitStatusCode(err error) (s *ExitStatus) {
	s = &ExitStatus{-1, err}
	if err == nil {
		return
	}

	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return
	}
	status, ok := exiterr.Sys().(syscall.WaitStatus)
	if !ok {
		return
	}

	s.code = status.ExitStatus()
	s.err = nil

	return
}
