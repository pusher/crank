package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Process struct {
	*EventLoop
	proto        *Prototype
	exitHandlers []func()
	_sendSignal  chan syscall.Signal
	outw         *os.File
	inr          *os.File
	command      *exec.Cmd
}

func NewProcess(proto *Prototype) *Process {
	return &Process{
		EventLoop:    NewEventLoop(),
		proto:        proto,
		exitHandlers: make([]func(), 0),
		_sendSignal:  make(chan syscall.Signal),
	}
}

func (p *Process) Run() error {
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

	command := exec.Command(p.proto.cmd, p.proto.args...)
	p.command = command

	// Inherit the environment with which crank was run
	command.Env = os.Environ()
	command.Env = append(command.Env, "LISTEN_FDS=1")

	// Pass file descriptors to the process
	command.ExtraFiles = append(command.ExtraFiles, p.proto.fd) // 3: accept socket
	command.ExtraFiles = append(command.ExtraFiles, outr)       // 4: client recv pipe
	command.ExtraFiles = append(command.ExtraFiles, inw)        // 5: client send pipe

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

	// Goroutine catches process exit
	go func() {
		if err := command.Wait(); err == nil {
			log.Printf("[process] Exited cleanly")
		} else {
			if exiterr, ok := err.(*exec.ExitError); ok {
				// The program has exited with an exit code != 0
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					// Prints the cause of exit - either exit status or signal should be
					// != -1 (-1 means not exited or not signaled). See
					// http://golang.org/pkg/syscall/#WaitStatus
					log.Printf(
						"[process] Unclean exit: %v (exit status: %v, signal: %v)",
						err, status.ExitStatus(), int(status.Signal()),
					)
				} else {
					log.Printf("[process] Unsupported ExitError: %v", err)
				}
			} else {
				log.Printf("[process] Unexpected exit: %v", err)
			}
		}

		p.NextTick(func() {
			for _, f := range p.exitHandlers {
				f()
			}
			p.EventLoop.Stop()
		})
	}()

	// Main run loop for process
	p.EventLoop.Run()

	return nil
}

// StopAccepting send a SIGHUP signal to the process
func (p *Process) StopAccepting() {
	p.NextTick(func() {
		p.sendSignal(syscall.SIGHUP)
	})
}

func (p *Process) Accept() {
	log.Print("[process] WARN: Accept not implemented")
}

// Stop stops the process gracefully by first sending SIGTERM (indicating that connections should be closed gracefully), then by sending a second SIGTERM (indicating that connections should be closed forcibly), then finally by sending a SIGKILL
func (p *Process) Stop() {
	p.NextTick(func() {
		p.sendSignal(syscall.SIGTERM)
	})
	p.AddTimer(1*time.Second, func() {
		p.sendSignal(syscall.SIGTERM)
	})
	p.AddTimer(2*time.Second, func() {
		p.sendSignal(syscall.SIGKILL)
	})
}

// Register a function to be called when the process exists
func (p *Process) OnExit(f func()) {
	p.exitHandlers = append(p.exitHandlers, f)
}

func (p *Process) sendSignal(sig syscall.Signal) {
	log.Print("[process] Sending signal: ", sig)
	p.command.Process.Signal(sig)
}
