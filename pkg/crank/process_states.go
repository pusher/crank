package crank

import (
	"../devnull"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type ProcessState interface {
	String() string
	run(*Process) error
}

type NewProcessState func(*Process) ProcessState

type ProcessStateCommon struct {
	enteredAt time.Time
}

func (s *ProcessStateCommon) init() {
	s.enteredAt = time.Now()
}

// Used as an alternative to time.After() to never get a timeout on a
// channel select.
var neverChan <-chan time.Time

func init() {
	neverChan = make(chan time.Time)
}

// NEW

func PROCESS_NEW(p *Process) ProcessState {
	s := new(ProcessStateNew)
	s.init()
	return s
}

type ProcessStateNew struct {
	ProcessStateCommon
}

func (s *ProcessStateNew) String() string {
	return "NEW"
}

func (s *ProcessStateNew) run(p *Process) (err error) {
	// Make sure we don't keep there handles around
	defer p.notifySocket.Close()
	defer p.logFile.Close()

	// TODO: Allow command arguments
	command := exec.Command(p.config.Command)

	// Inherit the environment with which crank was run
	// TODO: Remove environment inheriting, set sensible defaults
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")
	command.Env = append(command.Env, "NOTIFY_FD=4")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.bindSocket)   // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, p.notifySocket) // 4: notify socket

	command.Stdin, err = devnull.File()
	if err != nil {
		return
	}
	command.Stdout = p.logFile
	command.Stderr = p.logFile

	// Start process
	if err = command.Start(); err != nil {
		return
	}
	p.Process = command.Process

	// Goroutine catches process exit
	go func() {
		err := command.Wait()
		p.exitEvent <- getExitStatusCode(err)
	}()

	p.changeState(PROCESS_STARTING)
	return
}

// STARTING

func PROCESS_STARTING(p *Process) ProcessState {
	s := new(ProcessStateStarting)
	s.init()
	return s
}

type ProcessStateStarting struct {
	ProcessStateCommon
}

func (s *ProcessStateStarting) String() string {
	return "STARTING"
}

func (s *ProcessStateStarting) run(p *Process) (err error) {
	var timeout <-chan time.Time

	if p.config.StartTimeout > 0 {
		delay := (p.config.StartTimeout * time.Millisecond) -
			time.Now().Sub(s.enteredAt)
		timeout = time.After(delay)
	} else {
		timeout = neverChan
	}

	select {
	case <-timeout:
		return fmt.Errorf("Process did not start in time")
	case <-p.readyEvent:
		p.changeState(PROCESS_READY)
	case <-p.exitEvent:
		p.changeState(PROCESS_STOPPED)
	case <-p.shutdownAction:
		p.changeState(PROCESS_STOPPING)
	}

	return
}

// READY

func PROCESS_READY(p *Process) ProcessState {
	s := new(ProcessStateReady)
	s.init()
	return s
}

type ProcessStateReady struct {
	ProcessStateCommon
}

func (s *ProcessStateReady) String() string {
	return "READY"
}

func (s *ProcessStateReady) run(p *Process) (err error) {
	select {
	case <-p.readyEvent:
		p.log("Process started twice, ignoring")
	case <-p.exitEvent:
		p.changeState(PROCESS_STOPPED)
	case <-p.shutdownAction:
		p.changeState(PROCESS_STOPPING)
	}
	return
}

// STOPPING

func PROCESS_STOPPING(p *Process) ProcessState {
	s := new(ProcessStateStopping)
	s.init()
	// Tell the process to shutdown
	p.Signal(syscall.SIGTERM)
	return s
}

type ProcessStateStopping struct {
	ProcessStateCommon
}

func (s *ProcessStateStopping) String() string {
	return "STOPPING"
}

func (s *ProcessStateStopping) run(p *Process) (err error) {
	var timeout <-chan time.Time

	if p.config.StopTimeout > 0 {
		delay := (p.config.StopTimeout * time.Millisecond) -
			time.Now().Sub(s.enteredAt)
		timeout = time.After(delay)
	} else {
		timeout = neverChan
	}

	select {
	case <-timeout:
		err = fmt.Errorf("Process did not stop in time")
	case <-p.exitEvent:
		p.changeState(PROCESS_STOPPED)
	case <-p.shutdownAction:
		p.log("Stopping in the stopping state, ignoring")
	}

	return
}

// STOPPED

var REACTOR_STOP = fmt.Errorf("REACTOR STOP")

func PROCESS_STOPPED(p *Process) ProcessState {
	s := new(ProcessStateStopped)
	s.init()
	return s
}

type ProcessStateStopped struct {
	ProcessStateCommon
}

func (s *ProcessStateStopped) String() string {
	return "STOPPED"
}

func (s *ProcessStateStopped) run(p *Process) error {
	return REACTOR_STOP
}
