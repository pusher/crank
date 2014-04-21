package crank

import (
	"fmt"
	"net/rpc"
	"syscall"
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
	Start    bool
	Current  bool
	Shutdown bool
	Pid      int
}

type processFilter func(*Supervisor) *Supervisor

type PsQuery struct {
	ProcessQuery
}

type PsReply struct {
	PS []*ProcessInfo
}

type ProcessInfo struct {
	Pid   int
	State string
}

func (self *API) Ps(query *PsQuery, reply *PsReply) error {
	ss := self.m.childs

	if query.Pid > 0 {
		ss = ss.choose(func(p *Supervisor) bool {
			return p.Pid() == query.Pid
		})
	}

	if query.Start || query.Current || query.Shutdown {
		ss = ss.choose(func(p *Supervisor) bool {
			if query.Start && (p.state == PROCESS_NEW || p.state == PROCESS_STARTING) {
				return true
			}
			if query.Current && (p.state == PROCESS_READY) {
				return true
			}
			if query.Shutdown && (p.state == PROCESS_STOPPING || p.state == PROCESS_STOPPED || p.state == PROCESS_FAILED) {
				return true
			}
			return false
		})
	}

	reply.PS = make([]*ProcessInfo, 0, ss.len())
	for s := range ss {
		reply.PS = append(reply.PS, &ProcessInfo{s.Pid(), s.state.String()})
	}

	fmt.Println(query, reply)
	return nil
}

type KillQuery struct {
	ProcessQuery
	Signal string
	Wait   bool
}

type KillReply struct {
}

func (self *API) Kill(query *KillQuery, reply *KillReply) (err error) {
	var sig syscall.Signal
	if query.Signal == "" {
		sig = syscall.SIGTERM
	} else {
		if sig, err = str2signal(query.Signal); err != nil {
			return
		}
	}

	var ss supervisorSet
	if query.Start || query.Current || query.Shutdown || query.Pid > 0 {
		ss = self.m.childs
	} else {
		// Empty set
		ss = EmptySupervisorSet
	}

	if query.Start || query.Current || query.Shutdown {
		ss = ss.choose(func(p *Supervisor) bool {
			if query.Start && (p.state == PROCESS_NEW || p.state == PROCESS_STARTING) {
				return true
			}
			if query.Current && (p.state == PROCESS_READY) {
				return true
			}
			if query.Shutdown && (p.state == PROCESS_STOPPING || p.state == PROCESS_STOPPED || p.state == PROCESS_FAILED) {
				return true
			}
			return false
		})
	}

	if query.Pid > 0 {
		ss = ss.choose(func(p *Supervisor) bool {
			return p.Pid() == query.Pid
		})
	}

	ss.each(func(s *Supervisor) {
		s.Signal(sig)
	})

	fmt.Println(query, reply)
	return
}

func processSelect(ps []*Supervisor, filter processFilter) []*Supervisor {
	var processes []*Supervisor
	for _, p := range ps {
		p2 := filter(p)
		if p2 != nil {
			processes = append(processes, p2)
		}
	}
	return processes
}
