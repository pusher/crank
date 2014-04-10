package main

import (
	"./pkg/netutil"
	"flag"
	"log"
	"syscall"
)

var (
	addr string
	conf string
)

func init() {
	flag.StringVar(&addr, "addr", "", "external address to bind (e.g. 'tcp://:80')")
	flag.StringVar(&conf, "conf", "", "path to the process config file")
}

func main() {
	flag.Parse()

	// TODO: If required should not be a flag?
	// TODO: refactor this
	if addr == "" {
		log.Fatal("Missing required flag: addr")
	}
	if conf == "" {
		log.Fatal("Missing required flag: conf")
	}

	socket, err := netutil.ListenFile(addr)
	if err != nil {
		log.Fatal("OOPS", err)
	}
	log.Print(socket)

	manager := NewManager(conf, socket)
	go manager.Run()

	go OnSignal(manager.Restart, syscall.SIGHUP)
	go OnSignal(manager.Shutdown, syscall.SIGTERM, syscall.SIGINT)

	manager.OnShutdown.Wait()

	log.Println("Bye!")
}
