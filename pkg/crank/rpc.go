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
	Start    bool
	Current  bool
	Shutdown bool
	Pid      int
}

type processFilter func(*Process) *Process

// START

type StartQuery struct {
	Command      string
	StartTimeout int
	StopTimeout  int
	Wait         bool
}

type StartReply struct {
}

func (self *API) Start(query *StartQuery, reply *StartReply) error {
	// FIXME: concurrency access
	config := self.m.config.clone()

	if query.Command != "" {
		config.Command = query.Command
	}

	if query.StartTimeout > 0 {
		config.StartTimeout = time.Duration(query.StartTimeout) * time.Millisecond
	}

	if query.StopTimeout > 0 {
		config.StopTimeout = time.Duration(query.StopTimeout) * time.Millisecond
	}

	// TODO: support the query.Wait flag

	self.m.Start(config)

	return nil
}

// PS

type PsQuery struct {
	ProcessQuery
}

type PsReply struct {
	PS []*ProcessInfo
}

type ProcessInfo struct {
	Pid   int
	State string
	Usage *syscall.Rusage
	Err   error
}

func (pi *ProcessInfo) String() string {
	since := func(tv syscall.Timeval) time.Duration {
		return time.Duration(tv.Nano())
	}

	if pi.Err != nil {
		return fmt.Sprintf("%d %s\n", pi.Pid, pi.State, pi.Err)
	} else {
		usage := pi.Usage
		return fmt.Sprintf("%d %s %v %v %v\n", pi.Pid, pi.State, since(usage.Utime), since(usage.Stime), ByteCount(usage.Maxrss))
	}
}

func (self *API) Ps(query *PsQuery, reply *PsReply) error {
	ss := self.m.childs

	if query.Pid > 0 {
		ss = ss.choose(func(p *Process, _ ProcessState) bool {
			return p.Pid() == query.Pid
		})
	}

	if query.Start || query.Current || query.Shutdown {
		ss = ss.choose(func(p *Process, state ProcessState) bool {
			if query.Start && (state == PROCESS_STARTING) {
				return true
			}
			if query.Current && (state == PROCESS_READY) {
				return true
			}
			if query.Shutdown && (state == PROCESS_STOPPING) {
				return true
			}
			return false
		})
	}

	reply.PS = make([]*ProcessInfo, 0, ss.len())
	for s, state := range ss {
		usage, err := s.Usage()
		reply.PS = append(reply.PS, &ProcessInfo{s.Pid(), state.String(), usage, err})
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

	var ss processSet
	if query.Start || query.Current || query.Shutdown || query.Pid > 0 {
		ss = self.m.childs
	} else {
		// Empty set
		ss = EmptyProcessSet
	}

	if query.Start || query.Current || query.Shutdown {
		ss = ss.choose(func(p *Process, state ProcessState) bool {
			if query.Start && (state == PROCESS_STARTING) {
				return true
			}
			if query.Current && (state == PROCESS_READY) {
				return true
			}
			if query.Shutdown && (state == PROCESS_STOPPING) {
				return true
			}
			return false
		})
	}

	if query.Pid > 0 {
		ss = ss.choose(func(p *Process, _ ProcessState) bool {
			return p.Pid() == query.Pid
		})
	}

	ss.each(func(s *Process) {
		s.Signal(sig)
	})

	fmt.Println(query, reply)
	return
}
