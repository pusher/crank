package main

import (
	"flag"
	"log"
)

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

	proc := NewProcess(cmd, args, external.fd)
	proc.onExit(func() {
		log.Print("Process exited - WHOOT")
	})
	proc.Start()

	ExitOnSignal()
}
