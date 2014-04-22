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
	processCount    int
	processEvent    chan ProcessEvent
	startAction     chan *ProcessConfig
	shutdownAction  chan bool
	childs          processSet
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
		processEvent:    make(chan ProcessEvent),
		startAction:     make(chan *ProcessConfig),
		childs:          make(processSet),
		startingTracker: NewTimeoutTracker(),
		stoppingTracker: NewTimeoutTracker(),
	}
	return manager
}

func (_ *Manager) log(format string, v ...interface{}) {
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
		case config := <-self.startAction:
			if self.shuttingDown {
				self.log("Ignore start, manager is shutting down")
				continue
			}
			if self.childs.starting() != nil {
				self.log("Ignore start, new process is already being started")
				continue
			}
			self.startNewProcess(config)
		case <-self.shutdownAction:
			if self.shuttingDown {
				self.log("Already shutting down")
				continue
			}
			self.shuttingDown = true
			self.childs.each(func(p *Process) {
				self.stopProcess(p)
			})
		// timeouts
		case process := <-self.startingTracker.timeoutNotification:
			self.log("Process did not start in time. pid=%s", process.Pid())
			process.Kill()
		case process := <-self.stoppingTracker.timeoutNotification:
			self.log("Process did not stop in time. pid=%s", process.Pid())
			process.Kill()
		// process state transitions
		case e := <-self.processEvent:
			switch event := e.(type) {
			case *ProcessReadyEvent:
				process := event.process
				self.startingTracker.Remove(process)
				if process != self.childs.starting() {
					fail("Some other process is ready")
					continue
				}
				self.log("Process is ready %s", process)
				current := self.childs.ready()
				if current != nil {
					self.log("Shutting down the current process %s", current)
					self.stopProcess(current)
				}
				err := process.config.save(self.configPath)
				if err != nil {
					self.log("Failed saving the config: %s", err)
				}
				self.childs.updateState(process, PROCESS_READY)
			case *ProcessExitEvent:
				process := event.process

				self.startingTracker.Remove(process)
				self.stoppingTracker.Remove(process)
				self.childs.rem(process)

				self.log("Process exited. %s code=%d err=%s", process, event.code, event.err)

				if self.childs.len() == 0 {
					goto exit
				}
			default:
				fail("Unknown event: ", e)
			}
		}
	}

exit:

	// Cleanup
	self.childs.each(func(p *Process) {
		p.Kill()
	})
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Reload() {
	self.Start(self.config)
}

func (self *Manager) Start(c *ProcessConfig) {
	self.startAction <- c
}

func (self *Manager) Shutdown() {
	self.shutdownAction <- true
}

func (self *Manager) startNewProcess(config *ProcessConfig) {
	self.log("Starting a new process: %s", config)
	self.processCount += 1
	process, err := startProcess(self.processCount, config, self.socket, self.processEvent)
	if err != nil {
		self.log("Failed to start the process", err)
		return
	}
	self.childs.add(process)
	self.startingTracker.Add(process, process.config.StartTimeout)
}

func (self *Manager) stopProcess(process *Process) {
	if self.childs[process] == PROCESS_STOPPING {
		return
	}
	process.Shutdown()
	self.stoppingTracker.Add(process, process.config.StopTimeout)
	self.childs.updateState(process, PROCESS_STOPPING)
}
