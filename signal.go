package main

import (
	"os"
	"os/signal"
)

func OnSignal(f func(), signals ...os.Signal) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, signals...)
	for {
		<-c
		f()
	}
}
