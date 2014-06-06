package crank

import (
	"../devnull"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

func startProcess(id int, config *ProcessConfig, bindSocket *os.File, events chan<- Event) (p *Process, err error) {
	var (
		stdin        *os.File
		notifySocket *os.File
		logFile      *os.File
		ready        chan bool
	)

	ready = make(chan bool)

	lock := make(chan bool)
	defer close(lock)

	if stdin, err = devnull.File(); err != nil {
		return
	}

	if notifySocket, err = startProcessNotifier(ready); err != nil {
		return
	}
	defer notifySocket.Close()

	prefix := func() string {
		<-lock // once the channel is closed this will never block
		return fmt.Sprintf("%s %s ", time.Now().Format(time.StampMilli), p.String())
	}
	if logFile, err = startProcessLogger(os.Stdout, prefix); err != nil {
		return
	}
	defer logFile.Close()

	p = &Process{
		id:     id,
		config: config,
	}

	env := os.Environ()
	env = append(env, "LISTEN_FDS=1")
	env = append(env, "NOTIFY_FD=4")

	procAttr := os.ProcAttr{
		Dir: config.Cwd,
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
	command, err := exec.LookPath(config.Command[0])
	if err != nil {
		return nil, err
	}

	if p.Process, err = os.StartProcess(command, config.Command, &procAttr); err != nil {
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
	if p.Process == nil {
		return -1
	}

	return p.Process.Pid
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
