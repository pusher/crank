package crank

import (
	"fmt"
	"net/rpc"
	"syscall"
)

type RPC struct {
	*rpc.Server
	m *Manager
}

func NewRPC(m *Manager) *RPC {
	server := &RPC{rpc.NewServer(), m}
	err := server.RegisterName("crank", server)
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
	Start    *Supervisor
	Current  *Supervisor
	Shutdown []*Supervisor
}

func (self *RPC) Ps(query *PsQuery, reply *PsReply) error {
	all := !query.Start && !query.Current && !query.Shutdown

	var filterPid processFilter
	if query.Pid > 0 {
		filterPid = func(p *Supervisor) *Supervisor {
			if p == nil || p.Pid() != query.Pid {
				return nil
			}
			return p
		}
	} else {
		filterPid = func(p *Supervisor) *Supervisor {
			return p
		}
	}

	if query.Start || all {
		reply.Start = filterPid(self.m.starting)
	}
	if query.Current || all {
		reply.Current = filterPid(self.m.current)
	}
	if query.Shutdown || all {
		reply.Shutdown = processSelect(self.m.old.toArray(), filterPid)
	}

	fmt.Println(query, reply)
	return nil
}

type KillQuery struct {
	ProcessQuery
	Signal syscall.Signal
	Wait   bool
}

type KillReply struct {
}

func (self *RPC) Kill(query *KillQuery, reply *KillReply) (err error) {
	// TODO: By default don't kill any ?

	if query.Signal == 0 {
		query.Signal = syscall.SIGTERM
	}

	var processes []*Supervisor
	var filterPid processFilter
	if query.Pid > 0 {
		filterPid = func(p *Supervisor) *Supervisor {
			if p == nil || p.Pid() != query.Pid {
				return nil
			}
			return p
		}
	} else {
		filterPid = func(p *Supervisor) *Supervisor {
			return p
		}
	}

	appendProcess := func(p *Supervisor) {
		if p != nil {
			processes = append(processes, p)
		}
	}

	if query.Start {
		appendProcess(filterPid(self.m.starting))
	}
	if query.Current {
		appendProcess(filterPid(self.m.current))
	}
	if query.Shutdown {
		processes = append(processes, processSelect(self.m.old.toArray(), filterPid)...)
	}

	for _, p := range processes {
		p.Signal(query.Signal)
	}

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
