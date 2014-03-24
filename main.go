package main

import (
	"./logfile"
	"flag"
	"io"
	"log"
	"os"
	"syscall"
)

func main() {
	var addr = flag.String("addr", "", "external address to bind (e.g. ':80')")
	var outputFile = flag.String("out", "", "write stdout/err to file")
	flag.Parse()

	// TODO: If required should not be a flag?
	if len(*addr) == 0 {
		log.Fatal("Missing required flag: addr")
	}

	if flag.NArg() == 0 {
		log.Fatal("Missing COMMAND [Usage: crank OPTIONS COMMAND]")
	}

	cmd := flag.Arg(0)
	args := flag.Args()[1:]

	external, err := NewExternal(*addr)
	if err != nil {
		log.Fatal("OOPS", err)
	}
	log.Print(external)

	// Send process output to outputFile or stdout depending on whether flag passed
	var output io.Writer
	if len(*outputFile) > 0 {
		logOutput := logfile.New(*outputFile)
		go logOutput.Run()
		defer logOutput.Close()
		output = logOutput
	} else {
		output = os.Stdout
	}

	// Prototype is used to create new processes
	prototype := NewPrototype(cmd, args, external.Fd, output)

	manager := NewManager(prototype, 1)
	go manager.Run()

	// Restart processes on SIGHUP
	go OnSignalLoop(func() {
		manager.Restart()
	}, syscall.SIGHUP)

	ExitOnSignal()
}
