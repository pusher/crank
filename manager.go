package main

import (
	"log"
	"sync"
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
	restart        chan bool
	started        chan bool     // TODO pass PID
	exited         chan *Process
	newProcess     *Process
	currentProcess *Process
	oldProcesses   processSet
	OnShutdown     sync.WaitGroup
}

func NewManager(proto *Prototype, n int) *Manager {
	manager := &Manager{
		proto:        proto,
		restart:      make(chan bool),
		started:      make(chan bool),
		exited:       make(chan *Process),
		oldProcesses: make(processSet),
	}
	manager.OnShutdown.Add(1)
	return manager
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	if self.proto != nil {
		self.startNewProcess()
	}

	for {
		log.Print("[manager] For")
		select {
		case <-self.restart:
			log.Print("[manager] Restarting the process")
			self.startNewProcess()
			// TODO throttling
		case <-self.started:
			log.Printf("[manager] Process %d started", self.newProcess.Pid)
			if self.currentProcess != nil {
				log.Printf("[manager] Shutting down the current process %d", self.currentProcess.Pid)
				self.currentProcess.Shutdown()
				self.oldProcesses.Add(self.currentProcess)
			}
			self.currentProcess = self.newProcess
			self.newProcess = nil
		case process := <-self.exited:
			self.onProcessExit(process)
		}
	}
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Restart() {
	self.restart <- true
}

func (self *Manager) Shutdown() {
	if self.newProcess != nil {
		self.newProcess.Kill()
	}
	if self.currentProcess != nil {
		self.currentProcess.Kill()
	}
	for process, _ := range self.oldProcesses {
		process.Kill()
	}
	self.OnShutdown.Done()
}

func (self *Manager) startNewProcess() {
	log.Print("[manager] Starting a new process")
	if self.newProcess != nil {
		log.Print("[manager] New process is already being started")
		return // TODO what do we want to do in this case
	}
	self.newProcess = NewProcess(self.proto, self.started, self.exited)
	self.newProcess.Start()
}

func (self *Manager) onProcessExit(process *Process) {
	log.Printf("[manager] Process %d exited", process.Pid)
	// TODO process exit status?
	if process == self.newProcess {
		log.Print("[manager] Process exited in the new status")
		self.newProcess = nil
	} else if process == self.currentProcess {
		log.Print("[manager] Process exited in the current status")
		self.currentProcess = nil
		self.Shutdown()
		// TODO: shutdown
	} else {
		log.Print("[manager] Process exited in the old status")
		self.oldProcesses.Rem(process)
	}
}
