package main

import (
	"fmt"
	"syscall"
)

// TODO: Complete the table. It's not super solid since the signals are
//       os-dependent. It would be better to send strings and let crank do
//       the mapping.
var signalTable = map[string]syscall.Signal{
	"INT":  syscall.SIGINT,
	"TERM": syscall.SIGTERM,
	"KILL": syscall.SIGKILL,
	"HUP":  syscall.SIGHUP,
	"USR1": syscall.SIGUSR1,
	"USR2": syscall.SIGUSR2,
	"STOP": syscall.SIGSTOP,
	"CONT": syscall.SIGCONT,
}

type SignalValue struct {
	*syscall.Signal
}

func (self *SignalValue) Set(str string) error {
	if len(str) > 3 && str[:3] == "SIG" {
		str = str[3:]
	}

	sig, ok := signalTable[str]
	if !ok {
		return fmt.Errorf("Unknown signal %s", str)
	}
	(*self.Signal) = sig

	return nil
}

func (self *SignalValue) String() string {
	return self.Signal.String()
}
