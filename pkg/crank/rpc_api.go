package crank

import (
	"fmt"
	"net"
)

type RpcApi struct {
	m *Manager
}

func NewRpcApi(m *Manager) *RpcApi {
	return &RpcApi{m}
}

func (self *RpcApi) Echo(msg *string, reply *string) error {
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
