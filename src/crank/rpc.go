package crank

import (
	"fmt"
	"net/rpc"
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
	Cwd          string
	StartTimeout int
	StopTimeout  int
	Wait         bool
	Pid          int
}

type StartReply struct {
	Code int
}

func (self *API) Run(query *StartQuery, reply *StartReply) error {
	done := make(chan error, 1)
	self.m.actions <- &StartAction{query, reply, done}
	return <-done
}

// INFO

type InfoQuery struct{}

type InfoReply struct {
	Info *Info
}

func (self *API) Info(query *InfoQuery, reply *InfoReply) error {
	done := make(chan error, 1)
	self.m.actions <- &InfoAction{query, reply, done}
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
	Cid     int
	State   string
	Cwd     string
	Command []string
}

func (pi *ProcessInfo) String() string {
	return fmt.Sprintf("%d %d %s %#v %v", pi.Pid, pi.Cid, pi.State, pi.Cwd, pi.Command)
}

func (self *API) Ps(query *PsQuery, reply *PsReply) error {
	done := make(chan error, 1) // Make the reply async
	self.m.actions <- &PsAction{query, reply, done}
	return <-done
}

// KILL

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
