package main

import (
	"fmt"
	"net"
	"os"
)

// Binds to a TCP socket and makes it's Fd available for consumption.
// It would be used for example to pass into a new forked process.
type ExternalSocket struct {
	addr string
	File *os.File
}

func BindExternalSocket(addr string) (e *ExternalSocket, err error) {
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

	return &ExternalSocket{addr, file}, nil
}

func (e *ExternalSocket) String() string {
	return fmt.Sprintf("TCP socket listening on %v", e.addr)
}
