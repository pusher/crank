package crank

import (
	"fmt"
	"log"
	"os"
)

// Manager manages multiple process groups
type Manager struct {
	configPath      string
	config          *ProcessConfig
	socket          *os.File
	supervisorEvent chan *StateChangeEvent
	supervisorCount int
	restartAction   chan *ProcessConfig
	shutdownAction  chan bool
	childs          supervisorSet
	shuttingDown    bool
	startingTracker *TimeoutTracker
	stoppingTracker *TimeoutTracker
}

func NewManager(configPath string, socket *os.File) *Manager {
	config, err := loadProcessConfig(configPath)
	if err != nil {
		log.Println("Could not load config file: ", err)
	}

	manager := &Manager{
		configPath:      configPath,
		config:          config,
		socket:          socket,
		supervisorEvent: make(chan *StateChangeEvent),
		restartAction:   make(chan *ProcessConfig),
		childs:          make(supervisorSet),
		startingTracker: NewTimeoutTracker(),
		stoppingTracker: NewTimeoutTracker(),
	}
	return manager
}

func (self *Manager) log(format string, v ...interface{}) {
	log.Printf("[manager] "+format, v...)
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	if self.config != nil && self.config.Command != "" {
		self.startNewProcess(self.config)
	}

	go self.startingTracker.Run()
	go self.stoppingTracker.Run()

	for {
		select {
		// actions
		case c := <-self.restartAction:
			self.log("Restarting the process")
			self.startNewProcess(c)
		case <-self.shutdownAction:
			if self.shuttingDown {
				self.log("Already shutting down")
				continue
			}
			self.shuttingDown = true
			self.childs.each(func(s *Supervisor) {
				s.Shutdown()
			})
		// timeouts
		case s := <-self.startingTracker.timeoutNotification:
			s.err = fmt.Errorf("Process did not start in time")
			s.Kill()
		case s := <-self.stoppingTracker.timeoutNotification:
			s.err = fmt.Errorf("Process did not stop in time")
			s.Kill()
		// process state transitions
		case e := <-self.supervisorEvent:
			supervisor := e.supervisor
			switch e.state {
			case PROCESS_STARTING:
				self.startingTracker.Add(supervisor, supervisor.config.StartTimeout)
			case PROCESS_READY:
				self.startingTracker.Remove(supervisor)

				if supervisor != self.childs.starting() {
					fail("Some other process is ready")
					continue
				}
				self.log("Process %d is ready", supervisor.Pid())
				current := self.childs.current()
				if current != nil {
					self.log("Shutting down the current process %d", current.Pid())
					current.Shutdown()
				}
				err := supervisor.config.save(self.configPath)
				if err != nil {
					self.log("Failed saving the config: %s", err)
				}
			case PROCESS_STOPPING:
				self.stoppingTracker.Add(supervisor, supervisor.config.StopTimeout)
			case PROCESS_STOPPED, PROCESS_FAILED:
				self.startingTracker.Remove(supervisor)
				self.stoppingTracker.Remove(supervisor)

				allGone := self.onProcessExit(supervisor)
				if allGone {
					goto exit
				}
			}
			self.childs.updateState(supervisor, e.state)
		}
	}

exit:

	// Cleanup
	self.childs.each(func(s *Supervisor) {
		s.Kill()
	})
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
	if self.childs.starting() != nil {
		self.log("Ignore, new process is already being started")
		return
	}
	self.supervisorCount += 1
	s := NewSupervisor(self.supervisorCount, c, self.socket, self.supervisorEvent)
	go s.run()
	self.childs.add(s)
}

func (self *Manager) onProcessExit(s *Supervisor) bool {
	self.log("Process %d exited", s.Pid())

	self.childs.rem(s)
	return self.childs.len() == 0
}
