package crank

import (
	"log"
	"os"
	"sync"
)

type processSet map[*Process]bool

func (self processSet) add(p *Process) {
	self[p] = true
}

func (self processSet) rem(p *Process) {
	delete(self, p)
}

func (self processSet) toArray() []*Process {
	ary := make([]*Process, len(self))
	i := 0
	for v, _ := range self {
		ary[i] = v
		i += 1
	}
	return ary
}

// Manager manages multiple process groups
type Manager struct {
	configPath          string
	config              *ProcessConfig
	socket              *os.File
	processNotification chan *Process
	restartAction       chan bool
	newProcess          *Process
	currentProcess      *Process
	oldProcesses        processSet
	OnShutdown          sync.WaitGroup
	shuttingDown        bool
}

func NewManager(configPath string, socket *os.File) *Manager {
	config, err := loadProcessConfig(configPath)
	if err != nil {
		// TODO handle empty files as in the design
		log.Fatal(err)
	}

	manager := &Manager{
		configPath:          configPath,
		config:              config,
		socket:              socket,
		processNotification: make(chan *Process),
		restartAction:       make(chan bool),
		oldProcesses:        make(processSet),
	}
	manager.OnShutdown.Add(1)
	return manager
}

func (self *Manager) log(format string, v ...interface{}) {
	log.Printf("[manager] "+format, v...)
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	log.Println("Running the manager")

	// TODO this goroutine never terminates
	go func() {
		for {
			<-self.restartAction
			self.log("Restarting the process")
			self.startNewProcess()
		}
	}()

	if self.config != nil {
		self.restartAction <- true
	}

	for {
		p := <-self.processNotification
		switch p.StateName() {
		case "READY":
			if p != self.newProcess {
				panic("[manager] BUG, some other process is ready")
			}
			self.log("Process %d is ready", p.Pid)
			if self.currentProcess != nil {
				self.log("Shutting down the current process %d", self.currentProcess.Pid)
				self.currentProcess.Shutdown()
				self.oldProcesses.add(self.currentProcess)
			}
			self.currentProcess = self.newProcess
			self.newProcess = nil
			self.currentProcess.config.save(self.configPath)
		case "STOPPED":
			self.onProcessExit(p)
		}
	}
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Restart() {
	self.restartAction <- true
}

func (self *Manager) Shutdown() {
	if self.shuttingDown {
		self.log("Trying to shutdown twice !")
		return
	}
	self.shuttingDown = true
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
	self.log("Starting a new process")
	if self.newProcess != nil {
		self.log("New process is already being started")
		return // TODO what do we want to do in this case
	}
	self.newProcess = newProcess(self.config, self.socket, self.processNotification)
	self.newProcess.Start()
}

func (self *Manager) onProcessExit(p *Process) {
	self.log("Process %d exited", p.Pid)
	// TODO process exit status?
	if p == self.newProcess {
		self.log("Process exited in the new status")
		self.newProcess = nil
	} else if p == self.currentProcess {
		self.log("Process exited in the current status")
		self.currentProcess = nil
		self.Shutdown()
		// TODO: shutdown
	} else {
		self.log("Process exited in the old status")
		self.oldProcesses.rem(p)
	}
}
