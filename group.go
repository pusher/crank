package main

import (
	"fmt"
	"log"
	"time"
)

type Group struct {
	Id int
	*EventLoop
	createdAt       time.Time
	proto           *Prototype
	targetProcesses int
	startingSet     processSet
	acceptingSet    processSet
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

func NewGroup(id int, proto *Prototype, n int) *Group {
	return &Group{
		Id:              id,
		EventLoop:       NewEventLoop(),
		createdAt:       time.Now(),
		proto:           proto,
		targetProcesses: n,
		startingSet:     make(processSet),
		acceptingSet:    make(processSet),
		stoppingSet:     make(processSet),
	}
}

func (self *Group) String() string {
	// const layout = "2006-01-02@15:04:05"
	// return fmt.Sprintf("[group %v] ", self.createdAt.Format(layout))
	return fmt.Sprintf("[group:%v] ", self.Id)
}

func (self *Group) stateReport() string {
	return fmt.Sprintf("Processes: %v (Accepting: %v [target:%v], Starting: %v, Stopping: %v)", self.totalCount(), self.acceptingSet.Size(), self.targetProcesses, self.startingSet.Size(), self.stoppingSet.Size())
}

func (self *Group) Run() {
	self.scheduleThink()
	self.EventLoop.Run(time.Second, self.think)
	// TODO: Stop event loop when process exits
}

func (self *Group) Stop() {
	self.EventLoop.Stop()
	log.Print(self, "Group terminated")
}

// Reduce reduces the number of running processes in the group by 1
func (self *Group) Reduce() {
	// Reduce the target count by 1
	self.targetProcesses -= 1

	self.scheduleThink()
}

func (self *Group) Increase() {
	self.targetProcesses += 1

	self.scheduleThink()
}

// func (self *Group) IncrementProcesses(i int) {
// 	self.targetProcesses = self.targetProcesses + i
// 	self.scheduleThink()
// }

func (self *Group) scheduleThink() {
	self.NextTick(func() {
		self.think()
	})
}

// Aim here is to do one operation per tick
func (self *Group) think() {
	log.Print(self, self.stateReport())

	if self.targetProcesses == 0 && self.totalCount() == 0 {
		log.Print(self, "Terminating group (last process has exited)")
		self.Stop()
	} else if self.nonStoppingCount() > self.targetProcesses {
		log.Print(self, "Shutting down a process")
		p := self.acceptingSet.GetRand()
		p.Shutdown()
	} else if self.nonStoppingCount() < self.targetProcesses {
		log.Print(self, "Starting a new process")
		self.startProcess()
	}
}

func (self *Group) nonStoppingCount() int {
	return (self.startingSet.Size() + self.acceptingSet.Size())
}

func (self *Group) totalCount() int {
	return (self.startingSet.Size() + self.acceptingSet.Size() + self.stoppingSet.Size())
}

func (self *Group) startProcess() {
	onStarted := make(chan bool)

	process := NewProcess(self.proto, self.Id, onStarted)

	// Process is initally placed in the starting set
	self.startingSet.Add(process)

	// Once the process has been marked as started it means it's accepting
	go func() {
		<-onStarted
		self.NextTick(func() {
			self.startingSet.Rem(process)
			self.acceptingSet.Add(process)
		})
	}()

	// Remove process from all sets on exit
	process.OnExit(func() {
		self.NextTick(func() {
			self.startingSet.Rem(process)
			self.acceptingSet.Rem(process)
			self.stoppingSet.Rem(process)
		})
		self.scheduleThink()
	})

	process.Start()
}
