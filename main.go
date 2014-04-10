package main

import (
	"flag"
	"log"
	"syscall"
)

func main() {
	var addr = flag.String("addr", "", "external address to bind (e.g. ':80')")
	var configPath = flag.String("conf", "", "path to the process config file")
	flag.Parse()

	// TODO: If required should not be a flag?
	// TODO: refactor this
	if len(*addr) == 0 {
		log.Fatal("Missing required flag: addr")
	}
	if len(*configPath) == 0 {
		log.Fatal("Missing required flag: conf")
	}

	external, err := NewExternal(*addr)
	if err != nil {
		log.Fatal("OOPS", err)
	}
	log.Print(external)

	manager := NewManager(*configPath, external)
	go manager.Run()

	// Restart processes on SIGHUP
	go OnSignalLoop(func() {
		manager.Restart()
	}, syscall.SIGHUP)

	go OnSignalLoop(func() {
		manager.Shutdown()
	}, syscall.SIGTERM)

	manager.OnShutdown.Wait()
}
