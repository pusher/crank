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
