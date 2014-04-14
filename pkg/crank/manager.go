package crank

import (
	"log"
	"os"
	"sync"
)

type processSet map[*Process]bool

func (self processSet) Add(p *Process) {
	self[p] = true
}

func (self processSet) Rem(p *Process) {
	delete(self, p)
}

func (self processSet) ToArray() []*Process {
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
	configPath     string
	config         *ProcessConfig
	socket         *os.File
	restart        chan bool
	ready          chan bool // TODO pass PID
	exited         chan *Process
	newProcess     *Process
	currentProcess *Process
	oldProcesses   processSet
	OnShutdown     sync.WaitGroup
	shuttingDown   bool
}

func NewManager(configPath string, socket *os.File) *Manager {
	config, err := LoadProcessConfig(configPath)
	if err != nil {
		// TODO handle empty files as in the design
		log.Fatal(err)
	}

	manager := &Manager{
		configPath:   configPath,
		config:       config,
		socket:       socket,
		restart:      make(chan bool),
		ready:        make(chan bool),
		exited:       make(chan *Process),
		oldProcesses: make(processSet),
	}
	manager.OnShutdown.Add(1)
	return manager
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	if self.config != nil {
		self.startNewProcess()
	}

	for {
		select {
		case <-self.restart:
			log.Print("[manager] Restarting the process")
			self.startNewProcess()
			// TODO what's happening? what should we do?
		case <-self.ready:
			log.Printf("[manager] Process %d is ready", self.newProcess.Pid)
			if self.currentProcess != nil {
				log.Printf("[manager] Shutting down the current process %d", self.currentProcess.Pid)
				self.currentProcess.Shutdown()
				self.oldProcesses.Add(self.currentProcess)
			}
			self.currentProcess = self.newProcess
			self.newProcess = nil
			self.currentProcess.config.Save(self.configPath)
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
	if self.shuttingDown {
		log.Println("Trying to shutdown twice !")
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
	log.Print("[manager] Starting a new process")
	if self.newProcess != nil {
		log.Print("[manager] New process is already being started")
		return // TODO what do we want to do in this case
	}
	self.newProcess = NewProcess(self.config, self.socket, self.ready, self.exited)
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
