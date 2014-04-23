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
	events          chan Event
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
		events:          make(chan Event),
		startAction:     make(chan *ProcessConfig),
		shutdownAction:  make(chan bool),
		childs:          make(processSet),
		startingTracker: NewTimeoutTracker(),
		stoppingTracker: NewTimeoutTracker(),
	}
	return manager
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	self.startProcess(self.config)

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
			self.startProcess(config)
		case <-self.shutdownAction:
			if self.shuttingDown {
				self.log("Already shutting down")
				continue
			}
			self.log("Shutting down")
			self.shuttingDown = true
			self.childs.each(func(p *Process) {
				self.stopProcess(p)
			})
		// timeouts
		case process := <-self.startingTracker.timeoutNotification:
			self.plog(process, "Killing, did not start in time.")
			process.Kill()
		case process := <-self.stoppingTracker.timeoutNotification:
			self.plog(process, "Killing, did not stop in time.")
			process.Kill()
		// process state transitions
		case e := <-self.events:
			switch event := e.(type) {
			case *ProcessReadyEvent:
				process := event.process
				self.startingTracker.Remove(process)
				if process != self.childs.starting() {
					self.plog(process, "Oops, some other process is ready")
					continue
				}
				self.plog(process, "Process is ready")
				current := self.childs.ready()
				if current != nil {
					self.plog(current, "Shutting down old current")
					self.stopProcess(current)
				}
				self.config = process.config
				err := self.config.save(self.configPath)
				if err != nil {
					self.log("Failed saving the config: %s", err)
				}
				self.childs.updateState(process, PROCESS_READY)
			case *ProcessExitEvent:
				process := event.process

				self.startingTracker.Remove(process)
				self.stoppingTracker.Remove(process)
				self.childs.rem(process)

				self.plog(process, "Process exited. code=%d err=%v", event.code, event.err)

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

// Private methods

func (_ *Manager) log(format string, v ...interface{}) {
	log.Printf("[manager] "+format, v...)
}

func (m *Manager) plog(p *Process, format string, v ...interface{}) {
	args := make([]interface{}, 1, 1+len(v))
	args[0] = p
	args = append(args, v...)
	log.Printf("%s "+format, args...)
}

func (self *Manager) startProcess(config *ProcessConfig) {
	if config.Command == "" {
		self.log("Ignoring process start, command is missing")
		return
	}
	self.log("Starting a new process: %s", config)
	self.processCount += 1
	process, err := startProcess(self.processCount, config, self.socket, self.events)
	if err != nil {
		self.log("Failed to start the process", err)
		return
	}
	self.childs.add(process, PROCESS_STARTING)
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
