package main

import (
	"time"
)

type EventLoop struct {
	_exec chan func()
	_exit chan bool
}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		_exec: make(chan func(), 10),
		_exit: make(chan bool, 10),
	}
}

func (self *EventLoop) Run() {
	for {
		select {
		case f := <-self._exec:
			f()
		case <-self._exit:
			close(self._exit)
			close(self._exec)
			return
		}
	}
}

func (self *EventLoop) NextTick(f func()) {
	self._exec <- f
}

func (self *EventLoop) AddTimer(d time.Duration, f func()) {
	time.AfterFunc(d, func() {
		self._exec <- f
	})
}

func (self *EventLoop) Stop() {
	self._exit <- true
}
