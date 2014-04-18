package crank

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

// Base interface

type ProcessState func() (string, ProcessStateTransition)
type ProcessStateTransition func(*Supervisor) ProcessState

// States

func PROCESS_NEW() (string, ProcessStateTransition) {
	return "NEW", func(s *Supervisor) ProcessState {
		var err error
		s.process, err = startProcess(s.config.Command, s.config.Args, s.socket, s.readyEvent, s.exitEvent)
		if err != nil {
			return PROCESS_FAILED
		}
		return PROCESS_STARTING
	}
}

func PROCESS_STARTING() (string, ProcessStateTransition) {
	return "STARTING", func(s *Supervisor) ProcessState {
		var timeout <-chan time.Time

		if s.config.StartTimeout > 0 {
			delay := s.config.StartTimeout - time.Now().Sub(s.lastTransition)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			fmt.Errorf("Process did not start in time")
			s.Kill()
			return PROCESS_FAILED
		case <-s.readyEvent:
			return PROCESS_READY
		case <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.Signal(syscall.SIGTERM)
			return PROCESS_STOPPING
		}
	}
}

func PROCESS_READY() (string, ProcessStateTransition) {
	return "READY", func(s *Supervisor) ProcessState {
		select {
		case <-s.readyEvent:
			s.log("Process started twice, ignoring")
			return PROCESS_READY // TODO ok or kill?
		case <-s.exitEvent:
			return PROCESS_FAILED
		case <-s.shutdownAction:
			s.Signal(syscall.SIGTERM)
			return PROCESS_STOPPING
		}
	}
}

func PROCESS_STOPPING() (string, ProcessStateTransition) {
	return "STOPPING", func(s *Supervisor) ProcessState {
		var timeout <-chan time.Time

		if s.config.StopTimeout > 0 {
			delay := s.config.StopTimeout - time.Now().Sub(s.lastTransition)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			fmt.Errorf("Process did not stop in time")
			s.Kill()
			return PROCESS_FAILED
		case <-s.exitEvent: // TODO: Record exit status
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.log("Stopping in the stopping state, ignoring")
			return PROCESS_STOPPING
		}
	}
}

func PROCESS_STOPPED() (string, ProcessStateTransition) {
	return "STOPPED", nil
}

func PROCESS_FAILED() (string, ProcessStateTransition) {
	return "FAILED", nil
}

// Reactor

type Supervisor struct {
	process *Process
	config  *ProcessConfig
	socket  *os.File
	// state
	state           ProcessState
	stateName       string
	stateTransition ProcessStateTransition
	lastTransition  time.Time
	// actions
	startAction    chan bool
	shutdownAction chan bool
	// process events
	readyEvent chan bool
	exitEvent  chan ExitStatus
	// process update notifications
	processNotification chan<- *Supervisor
}

func NewSupervisor(config *ProcessConfig, socket *os.File, processNotification chan<- *Supervisor) *Supervisor {
	return &Supervisor{
		config:              config,
		socket:              socket,
		state:               PROCESS_NEW,
		stateName:           "",
		stateTransition:     nil,
		lastTransition:      time.Now(),
		startAction:         make(chan bool),
		shutdownAction:      make(chan bool),
		readyEvent:          make(chan bool),
		exitEvent:           make(chan ExitStatus),
		processNotification: processNotification,
	}
}

func (s *Supervisor) run() {
	var oldStateName string
	for {
		oldStateName = s.stateName
		s.stateName, s.stateTransition = s.state()

		if oldStateName != s.stateName {
			s.lastTransition = time.Now()
		}

		s.log("Changed state")
		s.processNotification <- s

		if s.stateTransition != nil {
			s.state = s.stateTransition(s)
		} else {
			return
		}
	}
}

func (s *Supervisor) log(format string, v ...interface{}) {
	if s.process != nil {
		log.Printf("s:"+s.process.String()+format, v...)
	} else {
		log.Printf("s:[NIL] "+format, v...)
	}
}

func (s *Supervisor) Pid() int {
	if s.process != nil {
		return s.process.Pid
	} else {
		return -1
	}
}

func (s *Supervisor) Start() {
	s.startAction <- true
}

// Tell the process to stop itself. A maximum delay is defined by the
// StopTimeout process config.
func (s *Supervisor) Shutdown() {
	s.shutdownAction <- true
}

func (s *Supervisor) Signal(sig syscall.Signal) error {
	if s.process == nil {
		return fmt.Errorf("Process missing")
	}
	s.log("Sending signal: %v", sig)
	return s.process.Signal(sig)
}

func (s *Supervisor) Kill() error {
	return s.Signal(syscall.SIGKILL)
}
