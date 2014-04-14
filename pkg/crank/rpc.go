package crank

import (
	"fmt"
	"net"
)

type RPC struct {
	m *Manager
}

func NewRPC(m *Manager) *RPC {
	return &RPC{m}
}

func (self *RPC) Echo(msg *string, reply *string) error {
	fmt.Println("Got ", *msg)
	*reply = "Thanks !"
	return nil
}

func BindRPCSocket(path string) (l net.Listener, err error) {
	addr, err := net.ResolveUnixAddr("unix", path)
	if err != nil {
		return
	}
	l, err = net.ListenUnix("unix", addr)
	return
}
