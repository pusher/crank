package main

import (
	"../../pkg/crank"
	"../../pkg/netutil"
	"flag"
	"log"
	"net"
	"os"
	"syscall"
)

var (
	addr string
	conf string
	sock string
)

func init() {
	flag.StringVar(&addr, "addr", os.Getenv("CRANK_ADDR"), "external address to bind (e.g. 'tcp://:80')")
	flag.StringVar(&conf, "conf", os.Getenv("CRANK_CONF"), "path to the process config file")
	flag.StringVar(&sock, "sock", os.Getenv("CRANK_SOCK"), "rpc socket address")
}

func main() {
	flag.Parse()

	if addr == "" {
		log.Fatal("Missing required flag: addr")
	}
	if conf == "" {
		log.Fatal("Missing required flag: conf")
	}
	if sock == "" {
		log.Fatal("Missing required flag: sock")
	}

	socket, err := netutil.BindFile(addr)
	if err != nil {
		log.Fatal("addr socket failed: ", err)
	}

	// Make sure the path is writeable
	f, err := os.OpenFile(conf, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal("Could not open config file", conf, err)
	}
	f.Close()

	rpcFile, err := netutil.BindFile(sock)
	if err != nil {
		log.Fatal("run socket failed: ", err)
	}
	rpcListener, err := net.FileListener(rpcFile)
	if err != nil {
		log.Fatal("BUG(rpcListener) : ", err)
	}

	manager := crank.NewManager(conf, socket)
	go onSignal(manager.Reload, syscall.SIGHUP)
	go onSignal(manager.Shutdown, syscall.SIGTERM, syscall.SIGINT)

	rpc := crank.NewRPCServer(manager)
	go rpc.Accept(rpcListener)

	manager.Run() // Blocking

	// Shutdown
	os.Remove(rpcFile.Name())

	log.Println("Bye!")
}
