package crank

import (
	"log"
	"os"
	"sync"
)

type supervisorSet map[*Supervisor]bool

func (self supervisorSet) add(s *Supervisor) {
	self[s] = true
}

func (self supervisorSet) rem(s *Supervisor) {
	delete(self, s)
}

func (self supervisorSet) toArray() []*Supervisor {
	ary := make([]*Supervisor, len(self))
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
	processNotification chan *Supervisor
	restartAction       chan *ProcessConfig
	shutdownAction      chan bool
	starting            *Supervisor
	current             *Supervisor
	old                 supervisorSet
	OnShutdown          sync.WaitGroup
	shuttingDown        bool
}

func NewManager(configPath string, socket *os.File) *Manager {
	config, err := loadProcessConfig(configPath)
	if err != nil {
		log.Println("Could not load config file: ", err)
	}

	manager := &Manager{
		configPath:          configPath,
		config:              config,
		socket:              socket,
		processNotification: make(chan *Supervisor),
		restartAction:       make(chan *ProcessConfig),
		old:                 make(supervisorSet),
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

	if self.config != nil && self.config.Command != "" {
		self.startNewProcess(self.config)
	}

	for {
		select {
		case c := <-self.restartAction:
			self.log("Restarting the process")
			self.startNewProcess(c)
		case <-self.shutdownAction:
			if self.shuttingDown {
				self.log("Already shutting down")
				continue
			}
			self.shuttingDown = true
			if self.starting != nil {
				self.starting.Shutdown()
			}
			if self.current != nil {
				self.current.Shutdown()
			}
			for process, _ := range self.old {
				process.Shutdown()
			}
		case p := <-self.processNotification:
			switch p.stateName {
			case "READY":
				if p != self.starting {
					fail("some other process is ready")
					continue
				}
				self.log("Process %d is ready", p.Pid)
				if self.current != nil {
					self.log("Shutting down the current process %d", self.current.Pid)
					self.current.Shutdown()
					self.old.add(self.current)
				}
				self.current = self.starting
				self.starting = nil
				err := self.current.config.save(self.configPath)
				if err != nil {
					self.log("Failed saving the config: %s", err)
				}
			case "STOPPED", "FAILED":
				exit := self.onProcessExit(p)
				if exit {
					break
				}
			}
		}
	}

	// Cleanup
	if self.starting != nil {
		self.starting.Kill()
	}
	if self.current != nil {
		self.current.Kill()
	}
	for process, _ := range self.old {
		process.Kill()
	}

	self.OnShutdown.Done()
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Reload() {
	self.restartAction <- self.config
}

func (self *Manager) Restart(c *ProcessConfig) {
	self.restartAction <- c
}

func (self *Manager) Shutdown() {
	self.shutdownAction <- true
}

func (self *Manager) startNewProcess(c *ProcessConfig) {
	self.log("Starting a new process")
	if self.starting != nil {
		self.log("Ignore, new process is already being started")
		return
	}
	self.starting = NewSupervisor(c, self.socket, self.processNotification)
	go self.starting.run()
}

func (self *Manager) onProcessExit(s *Supervisor) bool {
	self.log("Process %d exited", s.Pid())
	// TODO process exit status?
	if s == self.starting {
		self.log("Process exited in the new status")
		self.starting = nil
	} else if s == self.current {
		self.log("Process exited in the current status")
		self.current = nil
		self.Shutdown()
		return true
	} else {
		self.log("Process exited in the old status")
		self.old.rem(s)
	}
	return false
}
