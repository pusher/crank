package crank

import (
	"../devnull"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

const (
	PROCESS_NEW = ProcessState(iota)
	PROCESS_STARTING
	PROCESS_READY
	PROCESS_STOPPING
	PROCESS_STOPPED
)

type ProcessState int

type ExitStatus struct {
	code int
	err  error
}

type Process struct {
	*os.Process
	state    ProcessState
	config   *ProcessConfig
	socket   *os.File
	notify   *os.File
	onReady  chan bool
	onExited chan *Process
	shutdown chan bool
}

func NewProcess(config *ProcessConfig, socket *os.File, ready chan bool, exited chan *Process) *Process {
	return &Process{
		state:    PROCESS_NEW,
		config:   config,
		socket:   socket,
		onReady:  ready,
		onExited: exited,
		shutdown: make(chan bool),
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
	command.ExtraFiles = append(command.ExtraFiles, p.socket)  // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, notifySnd) // 4: notify socket

	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()
	command.Stdin = devnull.File

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

	ready := make(chan bool)
	exited := make(chan *ExitStatus)
	never := make(chan time.Time)

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
				ready <- true
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
		// TODO handle timeouts correctly - don't reset on each for loop iteration
		for {
			var timeout <-chan time.Time
			switch p.state {
			case PROCESS_STARTING:
				if p.config.StartTimeout > 0 {
					timeout = time.After(p.config.StartTimeout * time.Millisecond)
				} else {
					timeout = never
				}

				select {
				case <-timeout:
					p.Log("Process did not start in time, killing")
					p.Kill()
				case <-ready:
					p.Log("Process transitioning to ready")
					p.state = PROCESS_READY
					p.onReady <- true
				case <-exited:
					p.Log("Process exited while starting")
					p.state = PROCESS_STOPPED
				case <-p.shutdown:
					p.Log("Stopping in the starting state, sending SIGTERM")
					p.sendSignal(syscall.SIGTERM)
					p.state = PROCESS_STOPPING
				}

			case PROCESS_READY:
				select {
				case <-ready:
					p.Log("Process started twice")
				case <-exited:
					p.Log("Process exited while running")
					p.state = PROCESS_STOPPED
				case <-p.shutdown:
					p.Log("Stopping in the running state, sending SIGTERM")
					p.sendSignal(syscall.SIGTERM)
					p.state = PROCESS_STOPPING
				}

			case PROCESS_STOPPING:
				if p.config.StopTimeout > 0 {
					timeout = time.After(p.config.StopTimeout * time.Millisecond)
				} else {
					timeout = never
				}

				select {
				case <-timeout:
					p.Log("Process did not stop in time, killing")
					p.Kill()
				case <-exited:
					p.state = PROCESS_STOPPED
				case <-p.shutdown:
					p.Log("Stopping in the stopping state, noop")
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

func (p *Process) Kill() {
	p.sendSignal(syscall.SIGKILL)
}

// Stop stops the process with increased aggressiveness
func (p *Process) Shutdown() {
	p.shutdown <- true
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
