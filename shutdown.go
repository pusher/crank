package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func ExitOnSignal() {
	// Handle termination
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c
	log.Print("[main] Exiting cleanly")
}
