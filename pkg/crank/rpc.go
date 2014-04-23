package crank

import (
	"fmt"
	"net/rpc"
	"syscall"
	"time"
)

type API struct {
	m *Manager
}

func NewRPCServer(m *Manager) *rpc.Server {
	server := rpc.NewServer()
	api := &API{m}
	err := server.RegisterName("crank", api)
	if err != nil {
		panic(err) // Coding error
	}
	return server
}

// Used by other query structs
type ProcessQuery struct {
	Starting bool
	Ready    bool
	Stopping bool
	Pid      int
}

type processFilter func(*Process) *Process

// START

type StartQuery struct {
	Command      []string
	StartTimeout int
	StopTimeout  int
	Wait         bool
}

type StartReply struct {
}

func (self *API) Start(query *StartQuery, reply *StartReply) error {
	done := make(chan error, 1)
	self.m.actions <- &StartAction{query, reply, done}
	return <-done
}

// PS

type PsQuery struct {
	ProcessQuery
}

type PsReply struct {
	PS []*ProcessInfo
}

type ProcessInfo struct {
	Pid     int
	State   string
	Command []string
	Usage   *syscall.Rusage
	Err     error
}

func (pi *ProcessInfo) String() string {
	since := func(tv syscall.Timeval) time.Duration {
		return time.Duration(tv.Nano())
	}

	if pi.Err != nil {
		return fmt.Sprintf("%d %s %v %v\n", pi.Pid, pi.State, pi.Command, pi.Err)
	} else {
		usage := pi.Usage
		return fmt.Sprintf("%d %s %v %v %v %v\n", pi.Pid, pi.State, pi.Command, since(usage.Utime), since(usage.Stime), ByteCount(usage.Maxrss))
	}
}

func (self *API) Ps(query *PsQuery, reply *PsReply) error {
	done := make(chan error, 1) // Make the reply async
	self.m.actions <- &PsAction{query, reply, done}
	return <-done
}

type KillQuery struct {
	ProcessQuery
	Signal string
	Wait   bool
}

type KillReply struct{}

func (self *API) Kill(query *KillQuery, reply *KillReply) (err error) {
	done := make(chan error, 1)
	self.m.actions <- &KillAction{query, reply, done}
	return <-done
}
