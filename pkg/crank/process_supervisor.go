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
type ProcessStateTransition func(*ProcessSupervisor) ProcessState

// States

func PROCESS_NEW() (string, ProcessStateTransition) {
	return "NEW", func(s *ProcessSupervisor) ProcessState {
		var err error
		s.process, err = startProcess(s.config.Command, s.config.Args, s.socket, s.readyEvent, s.exitEvent)
		if err != nil {
			return PROCESS_FAILED
		}
		return PROCESS_STARTING
	}
}

func PROCESS_STARTING() (string, ProcessStateTransition) {
	return "STARTING", func(s *ProcessSupervisor) ProcessState {
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
	return "READY", func(s *ProcessSupervisor) ProcessState {
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
	return "STOPPING", func(s *ProcessSupervisor) ProcessState {
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

type ProcessSupervisor struct {
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
	processNotification chan<- *ProcessSupervisor
}

func NewProcessSupervisor(config *ProcessConfig, socket *os.File, processNotification chan<- *ProcessSupervisor) *ProcessSupervisor {
	return &ProcessSupervisor{
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

func (s *ProcessSupervisor) run() {
	for {
		s.stateName, s.stateTransition = s.state()

		s.log("Changed state")
		s.processNotification <- s

		if s.stateTransition != nil {
			// oldState := s.state
			s.state = s.stateTransition(s)
			// FIXME: Cannot compare functions
			// if oldState != s.state {
			// 	s.lastTransition = time.Now()
			// }
		} else {
			return
		}
	}
}

func (s *ProcessSupervisor) log(format string, v ...interface{}) {
	if s.process != nil {
		log.Printf("s:"+s.process.String()+format, v...)
	} else {
		log.Printf("s:[NIL] "+format, v...)
	}
}

func (s *ProcessSupervisor) Pid() int {
	if s.process != nil {
		return s.process.Pid
	} else {
		return -1
	}
}

func (s *ProcessSupervisor) Start() {
	s.startAction <- true
}

// Tell the process to stop itself. A maximum delay is defined by the
// StopTimeout process config.
func (s *ProcessSupervisor) Shutdown() {
	s.shutdownAction <- true
}

func (s *ProcessSupervisor) Signal(sig syscall.Signal) error {
	if s.process == nil {
		return fmt.Errorf("Process missing")
	}
	s.log("Sending signal: %v", sig)
	return s.process.Signal(sig)
}

func (s *ProcessSupervisor) Kill() error {
	return s.Signal(syscall.SIGKILL)
}
