package crank

type ProcessEvent interface {
}

type ProcessReadyEvent struct {
	process *Process
}

type ProcessExitEvent struct {
	process *Process
	code    int
	err     error
}
