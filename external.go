package main

import (
	"fmt"
	"log"
	"net"
	"os"
)

type External struct {
	addr    string
	tcpAddr *net.TCPAddr
	socket  *net.TCPListener
	fd      *os.File
}

func NewExternal(addr string) *External {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	socket, e := net.ListenTCP("tcp", tcpAddr)
	if e != nil {
		log.Fatal(e)
	}
	file, e := socket.File()
	if e != nil {
		log.Fatal(e)
	}

	return &External{addr, tcpAddr, socket, file}
}

func (e *External) String() string {
	return fmt.Sprintf("TCP socket listening on %v", e.addr)
}
