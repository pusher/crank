package crank

import (
	"../devnull"
	"fmt"
	"os"
	"syscall"
)

func startProcess(id int, config *ProcessConfig, bindSocket *os.File, events chan<- Event) (p *Process, err error) {
	var (
		stdin        *os.File
		notifySocket *os.File
		logFile      *os.File
		ready        chan bool
	)

	ready = make(chan bool)

	if stdin, err = devnull.File(); err != nil {
		return
	}

	if notifySocket, err = startProcessNotifier(ready); err != nil {
		return
	}
	defer notifySocket.Close()

	prefix := func() string {
		return p.String()
	}
	if logFile, err = startProcessLogger(os.Stdout, prefix); err != nil {
		return
	}
	defer logFile.Close()

	p = &Process{
		id:     id,
		config: config,
	}

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
	if p.Process, err = os.StartProcess(config.Command, config.Args, &procAttr); err != nil {
		return nil, err
	}

	// Goroutine catches process exit
	go func() {
		for {
			ps, err := p.Wait()
			// Make sure we don't shutdown if the process is paused
			if ps != nil && !ps.Exited() {
				continue
			}
			code, err2 := getExitStatusCode(ps, err)
			events <- &ProcessExitEvent{p, code, err2}
			return
		}
	}()

	// Goroutine that transforms ready events
	go func() {
		for {
			select {
			case v := <-ready:
				if !v { // Channel closed
					return
				}
				events <- &ProcessReadyEvent{p}
			}
		}
	}()

	return p, nil
}

type Process struct {
	*os.Process
	id     int
	config *ProcessConfig
}

func (p *Process) Pid() int {
	if p.Process != nil {
		return p.Process.Pid
	} else {
		return -1
	}
}

func (p *Process) String() string {
	return fmt.Sprintf("id=%d pid=%d", p.id, p.Pid())
}

func (p *Process) Shutdown() error {
	return p.Signal(syscall.SIGTERM)
}

func getExitStatusCode(ps *os.ProcessState, err error) (int, error) {
	if ps == nil || err != nil {
		return 0, err
	}

	status, ok := ps.Sys().(syscall.WaitStatus)
	if !ok {
		return 0, fmt.Errorf("BUG, not a syscall.WaitStatus")
	}

	return status.ExitStatus(), nil
}
