package main

import (
	"fmt"
	"log"
	"time"
)

type Group struct {
	*EventLoop
	createdAt       time.Time
	proto           *Prototype
	targetAccepting int
	targetProcesses int
	acceptingSet    processSet
	notAcceptingSet processSet
	stoppingSet     processSet
}

type processSet map[*Process]bool

func (self processSet) Add(p *Process) {
	self[p] = true
}

func (self processSet) Rem(p *Process) {
	delete(self, p)
}

// GetRand returns random element due to random ordering of range
func (self processSet) GetRand() *Process {
	var p *Process
	for p, _ = range self {
		break
	}
	return p
}

func (self processSet) Size() int {
	return len(self)
}

func NewGroup(proto *Prototype, n int) *Group {
	return &Group{
		EventLoop:       NewEventLoop(),
		createdAt:       time.Now(),
		proto:           proto,
		targetAccepting: n,
		targetProcesses: n,
		acceptingSet:    make(processSet),
		notAcceptingSet: make(processSet),
		stoppingSet:     make(processSet),
	}
}

func (self *Group) String() string {
	const layout = "2006-01-02@15:04:05"
	return fmt.Sprintf("[group %v] ", self.createdAt.Format(layout))
}

func (self *Group) Run() {
	self.scheduleThink()
	self.EventLoop.Run()
	// TODO: Stop event loop when process exits
}

func (self *Group) Stop() {
	self.EventLoop.Stop()
	log.Print(self, "Group terminated")
}

// Reduce reduces the number of running processes in the group by 1
func (self *Group) Reduce() {
	// Reduce the target count by 1
	self.targetAccepting = self.targetAccepting - 1
	self.targetProcesses = self.targetProcesses - 1

	self.scheduleThink()
}

func (self *Group) Increase() {
	self.targetAccepting = self.targetAccepting + 1
	self.targetProcesses = self.targetProcesses + 1

	self.scheduleThink()
}

// func (self *Group) IncrementAccept(i int) {
// 	self.targetAccepting = self.targetAccepting + i
// 	self.scheduleThink()
// }
//
// func (self *Group) IncrementProcesses(i int) {
// 	self.targetProcesses = self.targetProcesses + i
// 	self.scheduleThink()
// }

func (self *Group) scheduleThink() {
	self.NextTick(func() {
		self.think()
	})
}

func (self *Group) think() {
	if self.acceptingSet.Size() == self.targetAccepting && self.nonStoppingCount() == self.targetProcesses {
		if self.totalCount() == 0 {
			log.Print(self, "Terminating group (last process has exited)")
			self.Stop()
		}
		return
	}

	// Aim here is to do one operation per tick
	if self.acceptingSet.Size() < self.targetAccepting && self.notAcceptingSet.Size() > 0 {
		log.Print(self, "Transitioning a non-accepting process to accepting")
		p := self.notAcceptingSet.GetRand()
		p.StartAccepting()
		self.notAcceptingSet.Rem(p)
		self.acceptingSet.Add(p)
	} else if self.acceptingSet.Size() > self.targetAccepting {
		log.Print(self, "Transitioning an accepting process to non-accepting")
		p := self.acceptingSet.GetRand()
		p.StopAccepting()
		self.acceptingSet.Rem(p)
		self.notAcceptingSet.Add(p)
	} else if self.totalCount() < self.targetProcesses {
		log.Print(self, "Starting a new process")
		self.startProcess()
	} else if self.nonStoppingCount() > self.targetProcesses {
		if self.notAcceptingSet.Size() > 0 {
			log.Print(self, "Stopping a non-accepting process")
			p := self.notAcceptingSet.GetRand()
			p.Stop()
			self.notAcceptingSet.Rem(p)
			self.stoppingSet.Add(p)
		} else if self.acceptingSet.Size() > 0 {
			log.Print(self, "Stopping an accepting process, non non-accepting ones available")
			p := self.acceptingSet.GetRand()
			p.Stop()
			self.acceptingSet.Rem(p)
			self.stoppingSet.Add(p)
		}
	}

	// Schedule next check in 1s
	self.AddTimer(time.Second, func() {
		self.think()
	})
}

func (self *Group) nonStoppingCount() int {
	return (self.acceptingSet.Size() + self.notAcceptingSet.Size())
}

func (self *Group) totalCount() int {
	return (self.acceptingSet.Size() + self.notAcceptingSet.Size() + self.stoppingSet.Size())
}

func (self *Group) startProcess() {
	proc := NewProcess(self.proto)
	self.acceptingSet.Add(proc)
	proc.OnExit(func() {
		// TODO: This needs to go onto group goroutine
		delete(self.acceptingSet, proc)
		delete(self.notAcceptingSet, proc)
		delete(self.stoppingSet, proc)

		self.scheduleThink()
	})
	proc.Start()
	proc.StartAccepting()
}
