package crank

import (
	"../devnull"
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	*os.Process
	config       *ProcessConfig
	bindSocket   *os.File
	notifySocket *os.File
	logFile      *os.File
	supervisor   *ProcessSupervisor
}

func newProcess(config *ProcessConfig, socket *os.File, processNotification chan<- *Process) *Process {
	p := &Process{
		config:     config,
		bindSocket: socket,
	}

	p.supervisor = NewProcessSupervisor(p, PROCESS_NEW, processNotification)
	go p.supervisor.run()

	return p
}

func (p *Process) String() string {
	if p.Process != nil {
		return fmt.Sprintf("[%v %v] ", p.Pid, p.supervisor.stateName)
	} else {
		return fmt.Sprintf("[NIL %v] ", p.supervisor.stateName)
	}
}

func (p *Process) StateName() string {
	return p.supervisor.stateName
}

func (p *Process) Config() *ProcessConfig {
	return p.config
}

func (p *Process) Start() {
	p.supervisor.startAction <- true
}

// Tell the process to stop itself. A maximum delay is defined by the
// StopTimeout process config.
func (p *Process) Shutdown() {
	p.supervisor.shutdownAction <- true
}

func (p *Process) Kill() error {
	return p.Signal(syscall.SIGKILL)
}

func (p *Process) Signal(sig syscall.Signal) error {
	p.log("Sending signal: %v", sig)
	return p.Process.Signal(sig)
}

func (p *Process) log(format string, v ...interface{}) {
	log.Print(p.String(), fmt.Sprintf(format, v...))
}

func (p *Process) startNotifier() (err error) {
	fds, err := syscall.Socketpair(syscall.AF_LOCAL, syscall.SOCK_STREAM, 0)
	if err != nil {
		return
	}
	rcv := os.NewFile(uintptr(fds[0]), "notify:rcv") // File name is arbitrary
	p.notifySocket = os.NewFile(uintptr(fds[1]), "notify:snd")

	pn := newProcessNotifier(rcv, p.supervisor.readyEvent)
	go pn.run()

	return
}

func (p *Process) startLogAggregator() (err error) {
	rcv, snd, err := os.Pipe()
	if err != nil {
		return
	}
	p.logFile = snd

	// Write stdout & stderr to the
	processLog := newProcessLog(os.Stdout, p)
	go processLog.copy(rcv)
	return
}

func (p *Process) launch() (err error) {
	if err = p.startNotifier(); err != nil {
		return
	}
	defer p.notifySocket.Close()

	if err = p.startLogAggregator(); err != nil {
		return
	}
	defer p.logFile.Close()

	// TODO: Allow command arguments
	command := exec.Command(p.config.Command)

	// Inherit the environment with which crank was run
	// TODO: Remove environment inheriting, set sensible defaults
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")
	command.Env = append(command.Env, "NOTIFY_FD=4")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.bindSocket)   // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, p.notifySocket) // 4: notify socket

	command.Stdin, err = devnull.File()
	if err != nil {
		return err
	}
	command.Stdout = p.logFile
	command.Stderr = p.logFile

	// Start process

	err = command.Start()
	p.Process = command.Process

	if err != nil {
		return err
	}

	// Goroutine catches process exit
	go func() {
		err := command.Wait()
		p.supervisor.exitEvent <- getExitStatusCode(err)
	}()

	return
}

func (p *Process) stop() {
	p.Signal(syscall.SIGTERM)
}

type ExitStatus struct {
	code int
	err  error
}

func getExitStatusCode(err error) (s ExitStatus) {
	s = ExitStatus{-1, err}
	if err == nil {
		return
	}

	exiterr, ok := err.(*exec.ExitError)
	if !ok {
		return
	}
	status, ok := exiterr.Sys().(syscall.WaitStatus)
	if !ok {
		return
	}

	s.code = status.ExitStatus()
	s.err = nil

	return
}
