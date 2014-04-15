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

type processFilter func(*Process) *Process

type PsQuery struct {
	ProcessQuery
}

type PsReply struct {
	Start    *Process
	Current  *Process
	Shutdown []*Process
}

func (self *RPC) Ps(query *PsQuery, reply *PsReply) error {
	all := !query.Start && !query.Current && !query.Shutdown

	var filterPid processFilter
	if query.Pid > 0 {
		filterPid = func(p *Process) *Process {
			if p == nil || p.Pid != query.Pid {
				return nil
			}
			return p
		}
	} else {
		filterPid = func(p *Process) *Process {
			return p
		}
	}

	if query.Start || all {
		reply.Start = filterPid(self.m.newProcess)
	}
	if query.Current || all {
		reply.Current = filterPid(self.m.currentProcess)
	}
	if query.Shutdown || all {
		reply.Shutdown = processSelect(self.m.oldProcesses.toArray(), filterPid)
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

	var processes []*Process
	var filterPid processFilter
	if query.Pid > 0 {
		filterPid = func(p *Process) *Process {
			if p == nil || p.Pid != query.Pid {
				return nil
			}
			return p
		}
	} else {
		filterPid = func(p *Process) *Process {
			return p
		}
	}

	appendProcess := func(p *Process) {
		if p != nil {
			processes = append(processes, p)
		}
	}

	if query.Start {
		appendProcess(filterPid(self.m.newProcess))
	}
	if query.Current {
		appendProcess(filterPid(self.m.currentProcess))
	}
	if query.Shutdown {
		processes = append(processes, processSelect(self.m.oldProcesses.toArray(), filterPid)...)
	}

	for _, p := range processes {
		p.Signal(query.Signal)
	}

	fmt.Println(query, reply)
	return
}

func processSelect(ps []*Process, filter processFilter) []*Process {
	var processes []*Process
	for _, p := range ps {
		p2 := filter(p)
		if p2 != nil {
			processes = append(processes, p2)
		}
	}
	return processes
}
