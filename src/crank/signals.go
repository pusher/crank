package crank

import (
	"fmt"
	"strings"
	"syscall"
)

// TODO: Complete the table.
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

func str2signal(str string) (sig syscall.Signal, err error) {
	str2 := strings.ToUpper(str)
	if len(str2) > 3 && str2[:3] == "SIG" {
		str2 = str2[3:]
	}

	sig, ok := signalTable[str2]
	if !ok {
		err = fmt.Errorf("Unknown signal %s", str)
	}

	return
}
