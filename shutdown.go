package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
)

func ExitOnSignal() {
	OnSignal(func() {
		log.Print("[main] Exiting cleanly")
	}, os.Interrupt, syscall.SIGTERM)
}

func OnSignal(f func(), signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	<-c
	f()
}

func OnSignalLoop(f func(), signals ...os.Signal) {
	for {
		OnSignal(f, signals...)
	}
}
