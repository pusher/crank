package crank

type ProcessState int

const (
	PROCESS_STARTING = ProcessState(1 << iota)
	PROCESS_READY    = ProcessState(1 << iota)
	PROCESS_STOPPING = ProcessState(1 << iota)
)

func (ps ProcessState) String() string {
	switch ps {
	case PROCESS_STARTING:
		return "STARTING"
	case PROCESS_READY:
		return "READY"
	case PROCESS_STOPPING:
		return "STOPPING"
	default:
		return "BUG, unknown state"
	}
}
