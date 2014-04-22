package crank

import (
	"log"
	"os"
)

// Manager manages multiple process groups
type Manager struct {
	configPath      string
	config          *ProcessConfig
	socket          *os.File
	supervisorEvent chan *Supervisor
	restartAction   chan *ProcessConfig
	shutdownAction  chan bool
	childs          supervisorSet
	shuttingDown    bool
}

type StartupManager struct {
	// event interface
	startAction       chan *ProcessConfig
	startNotification chan *Supervisor
	shutdownAction		chan bool
	// internals
	starting          *Supervisor
}

func (self *StartupManager) Run() {
	for {
		if self.starting != nil {
			select {
			case config := <-self.startAction:
				if self.starting != nil {
					self.launch(config, self.stateEvent)
				}
			case state := <-self.stateEvent:
				// TODO check if starting process is the right one
				if state == PROCESS_READY {
					started := self.starting
					self.starting = nil
					self.startNotification <- started
				} else {
					self.starting = nil
					// TODO failureNotification?
				}
			case <-self.shutdownAction:
				if self.starting != nil {
					self.starting.Kill()
				}
			}
		}
	}
}

func (self *StartupManager) StartNewProcess(config *ProcessConfig) {
	self.startAction <- config
}

func (self *StartupManager) Shutdown() {
	self.shutdownAction <- true
}

func (self *StartupManager) launch(c *ProcessConfig, event chan(*Supervisor)) {
	self.log("Starting a new process")
	// Add the channel
	s := NewSupervisor(c, self.socket, self.supervisorEvent)
	go s.run()
}

type RunningManager struct {
	// event interface
	replaceAction       chan *Supervisor
	replaceNotification chan *Supervisor
	stoppedNotification	chan *Supervisor
	// internals
	current							*Supervisor
	stateEvent					chan *ProcessState
}

func (self *RunningManager) Run() {
	for {
		select {
		case newProcess := <-self.replaceAction:
			if newProcess != self.current {
				self.replaceNotification <- self.replaceProcess(newProcess)
			}
		case newState <- self.stateEvent:
				// TODO check if current process is the right one
			switch newState {
			case PROCESS_STOPPED, PROCESS_FAILED:
				stopped := self.current
				self.current = nil
				self.stoppedNotification <- stopped
			default:
				fail("Nobody expects the Spanish Inquisition!")
			}
		case <-self.shutdownAction:
			if self.current != nil {
				self.replaceNotification <- self.replaceProcess(nil)
			}
	}
}

func (self *RunningManager) Replace(s *Supervisor) {
	self.replaceAction <- s
}

func (self *RunningManager) Shutdown() {
	self.shutdownAction <- true
}

type ConfigManager struct {
	updateAction chan *Supervisor
}

func (self *ConfigManager) Run() {
	for {
		select {
		case config <- self.updateAction:
			err := config.save(self.configPath)
			if err != nil {
				self.log("Failed saving the config: %s", err)
			}
	}
}

func (self *RunningManager) Replace(s *Supervisor) {
	self.updateAction <- s
}

func (self *RunningManager) Shutdown() {
	// NOOP for now
}

type ShutdownManager struct {
	// event interface
	shutdownAction       chan *Supervisor
	shutdownNotification chan *Supervisor
	// internals
	shuttingDown				 map[*Supervisor]bool
	stoppedEvent				 chan *Supervisor
}

func (self *ShutdownManager) Run() {
	for {
		select {
		case runningProcess := <-self.shutdownAction:
			if self.shuttingDown[runningProcess] == false {
				self.shuttingDown[runningProcess] = true
				runningProcess.Shutdown()
			}
		case stoppedProcess := <-self.stoppedEvent:
			if self.shuttingDown[stoppedProcess] == true {
				delete(self.shuttingDown, stoppedProcess)
				self.shutdownNotification <- stoppedProcess
			}
	}
}

func (self *ShutdownManager) ShutdownProcess(s *Supervisor) {
	self.shutdownAction <- s
}

func (self *ShutdownManager) Shutdown() {
	// NOOP for now
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
		supervisorEvent: make(chan *Supervisor),
		restartAction:   make(chan *ProcessConfig),
		childs:          make(supervisorSet),
	}
	return manager
}

func (self *Manager) log(format string, v ...interface{}) {
	log.Printf("[manager] "+format, v...)
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	go self.startupManager.Run()
	go self.runningManager.Run()
	go self.shutdownManager.Run()
	go self.configManager.Run()

	if self.config != nil && self.config.Command != "" {
		self.startupManager.startAction <- true
	}

	for {
		select {
		case config := <-self.restartAction:
			self.log("Restarting the process")
			self.startupManager.StartNewProcess()
		case <-self.shutdownAction:
			self.log("Shutting down all processes")
			self.startupManager.Shutdown()
			self.runningManager.Shutdown()
			self.shutdownManager.Shutdown()
			self.configManager.Shutdown()
		case p := <- self.startupManager.startNotification:
			self.log("Replacing the running process")
			self.runningManager.Replace(p)
			self.configManager.Update(p.config)
		case p := <-self.runningManager.replaceNotification:
			self.shutdownManager.ShutdownProcess(p)
		case p := <-self.runningManager.stoppedNotification:
			// NOOP, process died
		case p := <-self.shutdownManager.shutdownNotification:
			// NOOP, process shut down
		}
	}

	// Cleanup
	// self.childs.each(func(s *Supervisor) {
	// 	s.Kill()
	// })
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

func (self *Manager) onProcessExit(s *Supervisor) bool {
	self.log("Process %d exited", s.Pid())

	self.childs.rem(s)
	return self.childs.len() == 0
}
