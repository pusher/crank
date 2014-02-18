package main

import (
	"log"
	"time"
)

// Manager manages multiple process groups
type Manager struct {
	proto       *Prototype
	targetCount int
	group       *Group
	_restart    chan (bool)
}

func NewManager(proto *Prototype, n int) *Manager {
	return &Manager{
		proto:       proto,
		targetCount: n,
		_restart:    make(chan bool),
	}
}

// Run starts the event loop for the manager process
func (self *Manager) Run() {
	self.start()
	for {
		select {
		case <-self._restart:
			log.Print("[manager] Restarting - replacing process group")

			// Create new process group
			newGroup := NewGroup(self.proto, 0)
			go newGroup.Run()

			// Reduce the size of the new group while increasing the new one
			for {
				// Increase size of existing
				newGroup.Increase()
				<-time.After(time.Second)

				// Reduce size of old
				self.group.Reduce()

				if self.group.targetProcesses == 0 {
					self.group = newGroup
					break
				} else {
					<-time.After(time.Second)
				}
			}
		}
	}
}

// Restart queues and starts excecuting a restart job to replace the old process group with a new one.
func (self *Manager) Restart() {
	self._restart <- true
}

// start starts an initial process group
func (self *Manager) start() {
	self.group = NewGroup(self.proto, self.targetCount)
	go self.group.Run()
}
