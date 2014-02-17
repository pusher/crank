package main

import (
	"log"
)

type Group struct {
	proto       *Prototype
	targetCount int
	set         processSet
}

type processSet map[*Process]bool

func (self processSet) Add(p *Process) {
	self[p] = true
}

func NewGroup(proto *Prototype, n int) *Group {
	return &Group{
		proto:       proto,
		targetCount: n,
		set:         make(processSet),
	}
}

func (self *Group) Run() {
	// Start targetCount processes
	for i := 0; i < self.targetCount; i++ {
		self.startProcess()
	}
}

func (self *Group) startProcess() {
	proc := NewProcess(self.proto)
	self.set.Add(proc)
	proc.OnExit(func() {
		log.Print("Exited, starting new process")
		delete(self.set, proc)
		self.startProcess()
	})
	proc.Start()
}
