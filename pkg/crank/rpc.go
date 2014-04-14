package crank

import (
	"fmt"
	"net/rpc"
)

type RPC struct {
	*rpc.Server
	m *Manager
}

func NewRPC(m *Manager) *RPC {
	server := &RPC{rpc.NewServer(), m}
	err := server.RegisterName("crank", server)
	if err != nil {
		panic(err)
	}
	return server
}

func (self *RPC) Echo(msg *string, reply *string) error {
	fmt.Println("Got ", *msg)
	*reply = "Thanks !"
	return nil
}

type PsQuery struct {
	Start    bool
	Current  bool
	Shutdown bool
}

type PsReply struct {
	Start    *Process
	Current  *Process
	Shutdown []*Process
}

func (self *RPC) Ps(query *PsQuery, reply *PsReply) error {
	all := !query.Start && !query.Current && !query.Shutdown

	if query.Start || all {
		reply.Start = self.m.newProcess
	}
	if query.Current || all {
		reply.Current = self.m.currentProcess
	}
	if query.Shutdown || all {
		reply.Shutdown = self.m.oldProcesses.ToArray()
	}
	return nil
}
