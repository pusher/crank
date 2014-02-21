package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

type Process struct {
	*EventLoop
	proto         *Prototype
	exitHandlers  []func()
	_sendSignal   chan syscall.Signal
	outw          *os.File
	inr           *os.File
	command       *exec.Cmd
	acceptingCond *sync.Cond
	accepting     bool
}

func NewProcess(proto *Prototype) *Process {
	return &Process{
		EventLoop:     NewEventLoop(),
		proto:         proto,
		exitHandlers:  make([]func(), 0),
		_sendSignal:   make(chan syscall.Signal),
		acceptingCond: sync.NewCond(new(sync.RWMutex)),
	}
}

func (p *Process) Start() {
	// Pipe for crank -> process
	outr, outw, err := os.Pipe()
	if err != nil {
		log.Print("Error creating pipe", err)
	}

	// Pipe for process -> crank
	inr, inw, err := os.Pipe()
	if err != nil {
		log.Print("Error creating pipe", err)
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
	}
	log.Print("[process] Process started")

	// Close unused pipe-ends
	outr.Close()
	inw.Close()
	p.outw = outw
	p.inr = inr

	// Read on pipe from child, and process commands
	go func() {
		data := make([]byte, 1024)
		var err error
		var n int
		var command string
		for {
			n, err = inr.Read(data)
			if err != nil || n == 0 {
				log.Print("[process] Error reading on pipe: ", err)
				return
			}

			if err, command, _ = decodePipeCommand(data[0:n]); err != nil {
				log.Printf("[process] Invalid data recd on pipe: ", err)
				return
			}

			log.Print("[process] Received command on pipe: ", command)

			switch command {
			case "NOW_ACCEPTING":
				p.acceptingCond.L.Lock()
				p.accepting = true
				p.acceptingCond.L.Unlock()
				p.acceptingCond.Broadcast()
			default:
				log.Print("[process] Unknown command received: ", command)
			}
		}
	}()

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
	go p.EventLoop.Run()
}

// StopAccepting send a SIGHUP signal to the process
func (p *Process) StopAccepting() {
	p.NextTick(func() {
		p.sendSignal(syscall.SIGHUP)
	})
}

// TODO: Do we really want this to be blocking?
func (p *Process) StartAccepting() {
	// Request that the process accepts
	p.NextTick(func() {
		p.sendPipeCommand("START_ACCEPTING", new(interface{}))
	})

	// Wait till NOW_ACCEPTING has been received from child
	// TODO: Astract this logic - it's a mess
	p.acceptingCond.L.Lock()
	for !p.accepting {
		p.acceptingCond.Wait()
	}
	p.acceptingCond.L.Unlock()
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

func (p *Process) sendPipeCommand(command string, args interface{}) {
	log.Printf("[process] Sending command on pipe: %v", command)
	json := fmt.Sprintf("[\"%v\", {}]", command)
	if _, err := p.outw.Write([]byte(json)); err != nil {
		log.Print("Error writing to outw: ", err)
	}
}

func decodePipeCommand(data []byte) (err error, command string, args interface{}) {
	var obj interface{}

	if err = json.Unmarshal(data, &obj); err != nil {
		return
	} else {
		switch arr := obj.(type) {
		default:
			err = errors.New("Invalid protocol")
			return
		case []interface{}:
			args = arr[1]

			switch c := arr[0].(type) {
			case string:
				command = c
				return
			default:
				return
			}
		}
	}
}
