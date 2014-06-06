package crank

type Action interface{}

type ShutdownAction bool

// RPC actions

type StartAction struct {
	query *StartQuery
	reply *StartReply
	done  chan<- error
}

type InfoAction struct {
	query *InfoQuery
	reply *InfoReply
	done  chan<- error
}

type PsAction struct {
	query *PsQuery
	reply *PsReply
	done  chan<- error
}

type KillAction struct {
	query *KillQuery
	reply *KillReply
	done  chan<- error
}
