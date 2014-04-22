package crank

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

// Base interface

type ProcessStateTransition func(*ProcessState, *Supervisor) *ProcessState
type ProcessState struct {
	name string
	run  ProcessStateTransition
}

func (ps *ProcessState) String() string {
	return ps.name
}

func (ps *ProcessState) next(s *Supervisor) *ProcessState {
	return ps.run(ps, s)
}

// States

var PROCESS_NEW = &ProcessState{
	"NEW",
	func(current *ProcessState, s *Supervisor) *ProcessState {
		if s.config == nil || s.config.Command == "" {
			s.err = fmt.Errorf("Config missing")
			return PROCESS_FAILED
		}

		s.process, s.err = startProcess(s.config.Command, s.config.Args, s.socket, s.readyEvent, s.exitEvent)
		if s.err != nil {
			return PROCESS_FAILED
		}
		return PROCESS_STARTING
	},
}

var PROCESS_STARTING = &ProcessState{
	"STARTING",
	func(current *ProcessState, s *Supervisor) *ProcessState {
		var timeout <-chan time.Time

		if s.config.StartTimeout > 0 {
			delay := s.config.StartTimeout - time.Now().Sub(s.lastTransition)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			s.err = fmt.Errorf("Process did not start in time")
			s.Kill()
			return PROCESS_FAILED
		case <-s.readyEvent:
			return PROCESS_READY
		case s.exitStatus = <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.Signal(syscall.SIGTERM)
			return PROCESS_STOPPING
		}
	},
}

var PROCESS_READY = &ProcessState{
	"READY",
	func(current *ProcessState, s *Supervisor) *ProcessState {
		select {
		case <-s.readyEvent:
			s.log("Process started twice, ignoring")
			return current
		case s.exitStatus = <-s.exitEvent:
			return PROCESS_FAILED
		case <-s.shutdownAction:
			s.Signal(syscall.SIGTERM)
			return PROCESS_STOPPING
		}
	},
}

var PROCESS_STOPPING = &ProcessState{
	"STOPPING",
	func(current *ProcessState, s *Supervisor) *ProcessState {
		var timeout <-chan time.Time

		if s.config.StopTimeout > 0 {
			delay := s.config.StopTimeout - time.Now().Sub(s.lastTransition)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			s.err = fmt.Errorf("Process did not stop in time")
			s.Kill()
			return PROCESS_FAILED
		case s.exitStatus = <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.log("Stopping in the stopping state, ignoring")
			return current
		}
	},
}

var PROCESS_STOPPED = &ProcessState{name: "STOPPED"}
var PROCESS_FAILED = &ProcessState{name: "FAILED"}

// Reactor

type StateChangeEvent struct {
	supervisor *Supervisor
	state      *ProcessState
}

type Supervisor struct {
	id      int
	process *Process
	config  *ProcessConfig
	socket  *os.File
	// state
	state          *ProcessState
	lastTransition time.Time
	err            error
	exitStatus     ExitStatus
	// actions
	shutdownAction chan bool
	// process events
	readyEvent chan bool
	exitEvent  chan ExitStatus
	// process update notifications
	supervisorEvent chan<- *StateChangeEvent
}

func NewSupervisor(id int, config *ProcessConfig, socket *os.File, supervisorEvent chan<- *StateChangeEvent) *Supervisor {
	return &Supervisor{
		id:              id,
		config:          config,
		socket:          socket,
		state:           PROCESS_NEW,
		lastTransition:  time.Now(),
		shutdownAction:  make(chan bool),
		readyEvent:      make(chan bool),
		exitEvent:       make(chan ExitStatus),
		supervisorEvent: supervisorEvent,
	}
}

func (s *Supervisor) run() {
	var newState *ProcessState
	for {
		if s.state.run == nil {
			return
		}

		newState = s.state.next(s)

		if newState == s.state {
			continue
		}

		s.log("Changing state from %s to %s", s.state, newState)
		s.lastTransition = time.Now()
		s.state = newState
		s.supervisorEvent <- &StateChangeEvent{s, newState}
	}
}

func (s *Supervisor) log(format string, v ...interface{}) {
	args := make([]interface{}, 2, 2+len(v))
	args[0] = s.id
	args[1] = s.process
	args = append(args, v...)
	log.Printf("[sid=%d %s state=%s]: "+format, args...)
}

func (s *Supervisor) Pid() int {
	if s.process != nil {
		return s.process.Pid
	} else {
		return -1
	}
}

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
