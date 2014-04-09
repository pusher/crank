package main

import (
	"log"
)

type processSet map[*Process]bool

func (self processSet) Add(p *Process) {
	self[p] = true
}

func (self processSet) Rem(p *Process) {
	delete(self, p)
}

// Manager manages multiple process groups
type Manager struct {
	proto          *Prototype
	restart        chan (bool)
	started        chan (bool) // TODO pass PID
	newProcess     *Process
	currentProcess *Process
	oldProcesses   processSet
}

func NewManager(proto *Prototype, n int) *Manager {
	return &Manager{
		proto:        proto,
		restart:      make(chan bool),
		oldProcesses: make(processSet),
	}
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	if self.proto != nil {
		self.startNewProcess()
	}

	for {
		select {
		case <-self.restart:
			log.Print("[manager] Restarting the process")
			self.startNewProcess()
			// TODO throttling
		case <-self.started:
			if self.currentProcess != nil {
				self.currentProcess.Shutdown()
				self.oldProcesses.Add(self.currentProcess)
			}
			self.currentProcess = self.newProcess
			self.newProcess = nil
		}
	}
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Restart() {
	self.restart <- true
}

func (self *Manager) startNewProcess() {
	if self.newProcess != nil {
		return // TODO what do we want to do in this case
	}
	self.newProcess = NewProcess(self.proto, self.started)
	self.newProcess.Start()
	self.newProcess.OnExit(self.onProcessExit)
}

func (self *Manager) onProcessExit(process *Process) {
	// TODO process exit status?
	if process == self.newProcess {
		log.Print("[manager] Process exited in the new status")
		self.newProcess = nil
	} else if process == self.currentProcess {
		log.Print("[manager] Process exited in the current status")
		self.currentProcess = nil
		// TODO: shutdown
	} else {
		self.oldProcesses.Rem(process)
	}
}
