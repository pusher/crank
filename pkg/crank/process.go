package crank

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	*os.Process
	state          ProcessState
	config         *ProcessConfig
	bindSocket     *os.File
	notifySocket   *os.File
	logFile        *os.File
	readyEvent     chan bool
	exitEvent      chan ExitStatus
	shutdownAction chan bool
	processChange  chan<- *Process
}

func newProcess(config *ProcessConfig, socket *os.File, processChange chan<- *Process) *Process {
	p := &Process{
		config:         config,
		bindSocket:     socket,
		processChange:  processChange,
		readyEvent:     make(chan bool),
		exitEvent:      make(chan ExitStatus), // TODO: Make use of those
		shutdownAction: make(chan bool),
	}
	p.state = PROCESS_NEW(p)
	return p
}

func (p *Process) String() string {
	return fmt.Sprintf("[%v %s] ", p.Pid, p.state)
}

func (p *Process) State() ProcessState {
	return p.state
}

func (p *Process) Config() *ProcessConfig {
	return p.config
}

func (p *Process) Kill() error {
	return p.Signal(syscall.SIGKILL)
}

// Tell the process to stop itself. A maximum delay is defined by the
// StopTimeout process config.
func (p *Process) Shutdown() {
	p.shutdownAction <- true
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

	pn := newProcessNotifier(rcv, p.readyEvent)
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

func (p *Process) runReactorLoop() {
	var err error

	for err == nil {
		err = p.state.run(p)
	}

	if err != REACTOR_STOP {
		p.log("ERROR: ", err)
		p.Kill()
	}
}

func (p *Process) start() (err error) {
	if err = p.startNotifier(); err != nil {
		return
	}

	if err = p.startLogAggregator(); err != nil {
		return
	}

	go p.runReactorLoop()
	return
}

func (p *Process) changeState(newState NewProcessState) {
	state := newState(p)
	p.log("Changing state from %s to %s", p.state, state)
	p.state = state
	p.processChange <- p
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
