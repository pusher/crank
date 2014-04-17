package crank

import (
	"fmt"
	"time"
)

// Used as an alternative to time.After() to never get a timeout on a
// channel select.
var neverChan <-chan time.Time

func init() {
	neverChan = make(chan time.Time)
}

// Base interface

type ProcessState func() (string, ProcessStateTransition)
type ProcessStateTransition func(*ProcessSupervisor) ProcessState

// States

func PROCESS_NEW() (string, ProcessStateTransition) {
	return "NEW", func(s *ProcessSupervisor) ProcessState {
		<-s.startAction
		err := s.process.launch()
		if err != nil {
			return PROCESS_STOPPED
		}
		return PROCESS_STARTING
	}
}

func PROCESS_STARTING() (string, ProcessStateTransition) {
	return "STARTING", func(s *ProcessSupervisor) ProcessState {
		var enteredAt = time.Now()
		var timeout <-chan time.Time

		if s.process.config.StartTimeout > 0 {
			delay := (s.process.config.StartTimeout * time.Millisecond) - time.Now().Sub(enteredAt)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			fmt.Errorf("Process did not start in time")
			return PROCESS_STOPPED // TODO ok or kill?
		case <-s.readyEvent:
			return PROCESS_READY
		case <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.process.stop()
			return PROCESS_STOPPING
		}
	}
}

func PROCESS_READY() (string, ProcessStateTransition) {
	return "READY", func(s *ProcessSupervisor) ProcessState {
		select {
		case <-s.readyEvent:
			s.process.log("Process started twice, ignoring")
			return PROCESS_READY // TODO ok or kill?
		case <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.process.stop()
			return PROCESS_STOPPING
		}
	}
}

func PROCESS_STOPPING() (string, ProcessStateTransition) {
	return "STOPPING", func(s *ProcessSupervisor) ProcessState {
		var enteredAt = time.Now()
		var timeout <-chan time.Time

		if s.process.config.StopTimeout > 0 {
			delay := (s.process.config.StopTimeout * time.Millisecond) - time.Now().Sub(enteredAt)
			timeout = time.After(delay)
		} else {
			timeout = neverChan
		}

		select {
		case <-timeout:
			fmt.Errorf("Process did not stop in time")
			s.process.Kill()
			return PROCESS_STOPPING
		case <-s.exitEvent:
			return PROCESS_STOPPED
		case <-s.shutdownAction:
			s.process.log("Stopping in the stopping state, ignoring")
			return PROCESS_STOPPING // TODO ok?
		}
	}
}

func PROCESS_STOPPED() (string, ProcessStateTransition) {
	return "STOPPED", nil
}

// Reactor

type ProcessSupervisor struct {
	process *Process
	// state
	state           ProcessState
	stateName       string
	stateTransition ProcessStateTransition
	// actions
	startAction    chan bool
	shutdownAction chan bool
	// process events
	readyEvent chan bool
	exitEvent  chan ExitStatus
	// process update notifications
	processNotification chan<- *Process
}

func NewProcessSupervisor(process *Process, state ProcessState, processNotification chan<- *Process) *ProcessSupervisor {
	return &ProcessSupervisor{
		process:             process,
		state:               state,
		stateName:           "",
		stateTransition:     nil,
		startAction:         make(chan bool),
		shutdownAction:      make(chan bool),
		readyEvent:          make(chan bool),
		exitEvent:           make(chan ExitStatus),
		processNotification: processNotification,
	}
}

func (self *ProcessSupervisor) run() {
	for {
		self.stateName, self.stateTransition = self.state()

		self.process.log("Changed state")
		self.processNotification <- self.process

		if self.stateTransition != nil {
			self.state = self.stateTransition(self)
		} else {
			return
		}
	}
}
