package crank

type Event interface{}

type ProcessReadyEvent struct {
	process *Process
}

type ProcessExitEvent struct {
	process *Process
	code    int
	err     error
}
