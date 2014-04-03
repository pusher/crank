package main

import (
	"time"
)

type EventLoop struct {
	exec chan Callback
	exit chan bool
}

type Callback func()

func NoopCallback() {}

func NewEventLoop() *EventLoop {
	return &EventLoop{
		exec: make(chan Callback, 10),
		exit: make(chan bool, 10),
	}
}

func (self *EventLoop) Run(every time.Duration, defaultCb Callback) {
	for {
		select {
		case cb := <-self.exec:
			cb()
		case <-time.After(every):
			defaultCb()
		case <-self.exit:
			close(self.exit)
			close(self.exec)
			return
		}
	}
}

func (self *EventLoop) NextTick(cb Callback) {
	self.exec <- cb
}

func (self *EventLoop) AddTimer(d time.Duration, cb Callback) {
	time.AfterFunc(d, func() {
		self.exec <- cb
	})
}

func (self *EventLoop) Stop() {
	self.exit <- true
}
