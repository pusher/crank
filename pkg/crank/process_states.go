package crank

type ProcessState string

func (ps ProcessState) String() string {
	return string(ps)
}

const (
	PROCESS_STARTING = ProcessState("STARTING")
	PROCESS_READY    = ProcessState("READY")
	PROCESS_STOPPING = ProcessState("STOPPING")
)
