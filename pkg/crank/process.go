package crank

import (
	"../devnull"
	"fmt"
	"os"
	"syscall"
)

func startProcess(name string, args []string, bindSocket *os.File, ready chan<- bool, exit chan<- ExitStatus) (p *Process, err error) {
	var (
		stdin        *os.File
		notifySocket *os.File
		logFile      *os.File
	)

	if stdin, err = devnull.File(); err != nil {
		return
	}

	if notifySocket, err = startProcessNotifier(ready); err != nil {
		return
	}
	defer notifySocket.Close()

	if logFile, err = startProcessLogger(os.Stdout, func() string { return p.String() }); err != nil {
		return
	}
	defer logFile.Close()

	p = &Process{}

	// TODO: Remove environment inheriting, set sensible defaults
	env := os.Environ()
	env = append(env, "LISTEN_FDS=1")
	env = append(env, "NOTIFY_FD=4")

	procAttr := os.ProcAttr{
		// TODO: Dir: dir,
		Env: env,
		Files: []*os.File{
			stdin,
			logFile,      // stdout
			logFile,      // stderr
			bindSocket,   // fd:3
			notifySocket, // fd:4
		},
	}

	// Start process
	if p.Process, err = os.StartProcess(name, args, &procAttr); err != nil {
		return
	}

	// Goroutine catches process exit
	go func() {
		for {
			ps, err := p.Wait()
			// Make sure we don't shutdown if the process is paused
			if ps != nil && !ps.Exited() {
				continue
			}
			exit <- getExitStatusCode(ps, err)
		}
	}()

	return p, nil
}

type Process struct {
	*os.Process
}

func (p *Process) String() string {
	if p.Process != nil {
		return fmt.Sprintf("[%d]", p.Pid)
	} else {
		return fmt.Sprintf("[NIL]")
	}
}

type ExitStatus struct {
	code int
	err  error
}

func getExitStatusCode(ps *os.ProcessState, err error) (s ExitStatus) {
	s = ExitStatus{-1, err}
	if ps == nil {
		return
	}

	status, ok := ps.Sys().(syscall.WaitStatus)
	if !ok {
		return
	}

	s.code = status.ExitStatus()
	s.err = nil

	return
}
