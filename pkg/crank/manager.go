package crank

import (
	"fmt"
	"log"
	"os"
	"syscall"
	"time"
)

// Manager manages multiple process groups
type Manager struct {
	configPath      string
	config          *ProcessConfig
	socket          *os.File
	processCount    int
	events          chan Event
	actions         chan Action
	childs          processSet
	shuttingDown    bool
	startingTracker *TimeoutTracker
	stoppingTracker *TimeoutTracker
	startingReply   *StartReply
	startingDone    chan<- error
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
		actions:         make(chan Action),
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
		case a := <-self.actions:
			switch action := a.(type) {
			case ShutdownAction:
				if self.shuttingDown {
					self.log("Already shutting down")
					continue
				}
				self.log("Shutting down")
				self.shuttingDown = true
				self.childs.each(func(p *Process) {
					self.stopProcess(p)
				})
			case *StartAction:
				query := action.query
				//reply := action.reply -- not used

				if self.shuttingDown {
					err := fmt.Errorf("Ignore start, manager is shutting down")
					self.log(err.Error())
					action.done <- err
					continue
				}
				if self.childs.starting() != nil {
					err := fmt.Errorf("Ignore start, new process is already being started")
					self.log(err.Error())
					action.done <- err
					continue
				}

				config := self.config.clone()

				if len(query.Command) > 0 {
					config.Command = query.Command
				}

				if query.StartTimeout > 0 {
					config.StartTimeout = time.Duration(query.StartTimeout) * time.Second
				}

				if query.StopTimeout > 0 {
					config.StopTimeout = time.Duration(query.StopTimeout) * time.Second
				}

				err := self.startProcess(config)
				if err != nil {
					action.done <- err
					continue
				}

				if query.Wait {
					self.log("RPC waiting for the process to start")
					self.startingReply = action.reply
					self.startingDone = action.done
				} else {
					action.done <- nil
				}
			case *PsAction:
				query := action.query
				reply := action.reply
				ps := self.childs

				if query.Pid > 0 {
					ps = ps.choose(func(p *Process, _ ProcessState) bool {
						return p.Pid() == query.Pid
					})
				}

				if query.Starting || query.Ready || query.Stopping {
					ps = ps.choose(func(p *Process, state ProcessState) bool {
						if query.Starting && (state == PROCESS_STARTING) {
							return true
						}
						if query.Ready && (state == PROCESS_READY) {
							return true
						}
						if query.Stopping && (state == PROCESS_STOPPING) {
							return true
						}
						return false
					})
				}

				reply.PS = make([]*ProcessInfo, 0, ps.len())
				for p, state := range ps {
					usage, err := p.Usage()
					reply.PS = append(reply.PS, &ProcessInfo{p.Pid(), state.String(), p.config.Command, usage, err})
				}

				action.done <- nil
			case *KillAction:
				query := action.query
				//reply := action.reply -- not used

				var sig syscall.Signal
				if query.Signal == "" {
					sig = syscall.SIGTERM
				} else {
					var err error
					if sig, err = str2signal(query.Signal); err != nil {
						action.done <- err
						continue
					}
				}

				var ps processSet
				if query.Starting || query.Ready || query.Stopping || query.Pid > 0 {
					ps = self.childs
				} else {
					// Empty set
					ps = EmptyProcessSet
				}

				if query.Starting || query.Ready || query.Stopping {
					ps = ps.choose(func(p *Process, state ProcessState) bool {
						if query.Starting && (state == PROCESS_STARTING) {
							return true
						}
						if query.Ready && (state == PROCESS_READY) {
							return true
						}
						if query.Stopping && (state == PROCESS_STOPPING) {
							return true
						}
						return false
					})
				}

				if query.Pid > 0 {
					ps = ps.choose(func(p *Process, _ ProcessState) bool {
						return p.Pid() == query.Pid
					})
				}

				ps.each(func(p *Process) {
					p.Signal(sig)
				})

				action.done <- nil
			default:
				fail("Unknown action: ", a)
			}
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

				if process == self.childs.starting() && self.startingReply != nil {
					self.startingDone <- nil
					self.startingReply = nil
					self.startingDone = nil
				}

				self.childs.updateState(process, PROCESS_READY)
			case *ProcessExitEvent:
				process := event.process

				self.startingTracker.Remove(process)
				self.stoppingTracker.Remove(process)

				if process == self.childs.starting() && self.startingReply != nil {
					self.startingReply.Code = event.code
					self.startingDone <- event.err
					self.startingReply = nil
					self.startingDone = nil
				}

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

func (self *Manager) SendAction(action Action) {
	self.actions <- action
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Reload() {
	done := make(chan error)
	self.SendAction(&StartAction{&StartQuery{}, nil, done})
	<-done
}

func (self *Manager) Shutdown() {
	self.SendAction(ShutdownAction(true))
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

func (self *Manager) startProcess(config *ProcessConfig) error {
	if len(config.Command) == 0 {
		self.log("Ignoring process start, command is missing")
		return fmt.Errorf("Command is missing")
	}

	self.log("Starting a new process: %s", config)
	self.processCount += 1
	process, err := startProcess(self.processCount, config, self.socket, self.events)
	if err != nil {
		self.log("Failed to start the process", err)
		return err
	}

	self.childs.add(process, PROCESS_STARTING)
	self.startingTracker.Add(process, process.config.StartTimeout)
	return nil
}

func (self *Manager) stopProcess(process *Process) {
	if self.childs[process] == PROCESS_STOPPING {
		return
	}
	process.Shutdown()
	self.stoppingTracker.Add(process, process.config.StopTimeout)
	self.childs.updateState(process, PROCESS_STOPPING)
}
