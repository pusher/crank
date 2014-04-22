package crank

import (
	"sync"
	"time"
)

type TimeoutTracker struct {
	timeouts            map[*Supervisor]time.Time
	ticker              *time.Ticker
	timeoutNotification chan (*Supervisor)
	stopAction          chan bool
	mutex               *sync.Mutex
}

func NewTimeoutTracker() *TimeoutTracker {
	return &TimeoutTracker{
		timeouts:            make(map[*Supervisor]time.Time),
		ticker:              time.NewTicker(100 * time.Millisecond),
		timeoutNotification: make(chan *Supervisor),
		stopAction:          make(chan bool),
		mutex:               &sync.Mutex{},
	}
}

func (self *TimeoutTracker) Add(p *Supervisor, timeout time.Duration) {
	if timeout <= 0 {
		return
	}
	self.mutex.Lock()
	self.timeouts[p] = time.Now().Add(time.Millisecond * timeout)
	self.mutex.Unlock()
}

func (self *TimeoutTracker) Remove(p *Supervisor) {
	self.mutex.Lock()
	delete(self.timeouts, p)
	self.mutex.Unlock()
}

func (self *TimeoutTracker) Run() {
	for {
		select {
		case t := <-self.ticker.C:
			self.expireOld(t)
		case <-self.stopAction:
			self.ticker.Stop()
			return
		}
	}
}

func (self *TimeoutTracker) Stop() {
	self.stopAction <- true
}

func (self *TimeoutTracker) expireOld(now time.Time) {
	self.mutex.Lock()
	for p, timeout := range self.timeouts {
		if timeout.Before(now) {
			delete(self.timeouts, p)
			self.timeoutNotification <- p
		}
	}
	self.mutex.Unlock()
}
