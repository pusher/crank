package main

import (
	"flag"
	"log"
	"os"
	"time"
)

func startProcess(cmd string, args []string, fd *os.File) {
	stopped := false

	proc := NewProcess(cmd, args, fd)
	proc.OnExit(func() {
		stopped = true
		log.Print("Process exited - WHOOT")
		go startProcess(cmd, args, fd)
	})
	proc.Start()

	<-time.After(2 * time.Second)
	if stopped {
		return
	}
	log.Print("Sending HUP")
	proc.SendHUP()

	<-time.After(2 * time.Second)
	if stopped {
		return
	}
	log.Print("Sending 1st TERM")
	proc.SendTerm()

	<-time.After(2 * time.Second)
	if stopped {
		return
	}
	log.Print("Sending 2nd TERM")
	proc.SendTerm()

	<-time.After(2 * time.Second)
	if stopped {
		return
	}
	log.Print("Sending KILL")
	proc.SendKill()
}

func main() {
	var addr = flag.String("addr", "", "external address to bind (e.g. ':80')")
	flag.Parse()

	// TODO: If required should not be a flag?
	if len(*addr) == 0 {
		log.Fatal("Missing required flag: addr")
	}

	if flag.NArg() < 1 {
		log.Print("Usage: crank OPTIONS COMMAND")
	}
	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	external := NewExternal(*addr)
	log.Print(external)

	go startProcess(cmd, args, external.fd)

	ExitOnSignal()
}
