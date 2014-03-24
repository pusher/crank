package main

import (
	"fmt"
	"net"
	"os"
)

// Binds to a TCP socket and makes it's Fd available for consumption.
// It would be used for example to pass into a new forked process.
type External struct {
	addr    string
	tcpAddr *net.TCPAddr
	socket  *net.TCPListener
	Fd      *os.File
}

func NewExternal(addr string) (e *External, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}
	socket, err := net.ListenTCP("tcp", tcpAddr)
	if err != nil {
		return
	}
	file, err := socket.File()
	if err != nil {
		return
	}

	return &External{addr, tcpAddr, socket, file}, nil
}

func (e *External) String() string {
	return fmt.Sprintf("TCP socket listening on %v", e.addr)
}
