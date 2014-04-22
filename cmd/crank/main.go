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
	run  string
)

func init() {
	flag.StringVar(&addr, "addr", os.Getenv("CRANK_ADDR"), "external address to bind (e.g. 'tcp://:80')")
	flag.StringVar(&conf, "conf", os.Getenv("CRANK_CONF"), "path to the process config file")
	flag.StringVar(&run, "run", os.Getenv("CRANK_RUN"), "rpc socket address")
}

func main() {
	flag.Parse()

	// TODO: If required should not be a flag?
	if addr == "" {
		log.Fatal("Missing required flag: addr")
	}
	if conf == "" {
		log.Fatal("Missing required flag: conf")
	}
	if run == "" {
		log.Fatal("Missing required flag: run")
	}

	socket, err := netutil.BindFile(addr)
	if err != nil {
		log.Fatal("addr socket failed: ", err)
	}

	// Make sure the path is writeable
	f, err := os.OpenFile(conf, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Fatal("Could not open config file at %s: %s", conf, err)
	}
	f.Close()

	rpcFile, err := netutil.BindFile(run)
	if err != nil {
		log.Fatal("run socket failed: ", err)
	}
	rpcListener, err := net.FileListener(rpcFile)
	if err != nil {
		log.Fatal("BUG(rpcListener) : ", err)
	}

	manager := crank.NewManager(conf, socket)

	rpc := crank.NewRPCServer(manager)

	go manager.Run()

	go onSignal(manager.Reload, syscall.SIGHUP)
	go onSignal(manager.Shutdown, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		manager.OnShutdown.Wait()
		rpcListener.Close()
		os.Remove(rpcFile.Name())
	}()

	rpc.Accept(rpcListener)

	log.Println("Bye!")
}
