package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type Process struct {
	cmd          string
	args         []string
	fd           *os.File
	exitHandlers []func()
	_sendSignal  chan syscall.Signal
	outw         *os.File
	inr          *os.File
}

func NewProcess(cmd string, args []string, fd *os.File) *Process {
	return &Process{
		cmd:          cmd,
		args:         args,
		fd:           fd,
		exitHandlers: make([]func(), 0),
		_sendSignal:  make(chan syscall.Signal),
	}
}

func (p *Process) Start() error {
	// Pipe for crank -> process
	outr, outw, err := os.Pipe()
	if err != nil {
		log.Print("Error creating pipe", err)
		return err
	}

	// Pipe for process -> crank
	inr, inw, err := os.Pipe()
	if err != nil {
		log.Print("Error creating pipe", err)
		return err
	}

	command := exec.Command(p.cmd, p.args...)

	// Inherit the environment with which crank was run
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.fd) // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, outr) // 4: client recv pipe
	command.ExtraFiles = append(command.ExtraFiles, inw)  // 5: client send pipe

	// TODO: Temporarily forward stdout & stderr
	stdout, _ := command.StdoutPipe()
	stderr, _ := command.StderrPipe()
	go io.Copy(os.Stdout, stdout)
	go io.Copy(os.Stderr, stderr)

	// Start process
	if err = command.Start(); err != nil {
		log.Fatal("Process start failed: ", err)
		return err
	}
	log.Print("[process] Process started")

	// Close unused pipe-ends
	outr.Close()
	inw.Close()
	p.outw = outw
	p.inr = inr

	_onexit := make(chan bool)

	// Goroutine catches process exit
	go func() {
		command.Wait()
		_onexit <- true

	}()

	// Main run loop for process
	go func() {
		for {
			select {
			case <-_onexit:
				log.Print("[process] Process exited")
				for _, f := range p.exitHandlers {
					f()
				}
			case sig := <-p._sendSignal:
				command.Process.Signal(sig)
			}
		}
	}()

	return nil
}

// Register a function to be called when the process exists
func (p *Process) OnExit(f func()) {
	p.exitHandlers = append(p.exitHandlers, f)
}

func (p *Process) SendHUP() {
	p._sendSignal <- syscall.SIGHUP
}

func (p *Process) SendTerm() {
	p._sendSignal <- syscall.SIGTERM
}

func (p *Process) SendKill() {
	p._sendSignal <- syscall.SIGKILL
}
