package crank

import (
	"fmt"
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
