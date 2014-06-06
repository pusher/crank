package crank

type processSet map[*Process]ProcessState

func (set processSet) add(s *Process, state ProcessState) {
	set[s] = state
}

func (set processSet) rem(s *Process) {
	delete(set, s)
}

func (set processSet) len() int {
	return len(set)
}

func (set processSet) updateState(s *Process, state ProcessState) {
	if _, ok := set[s]; !ok {
		fail("Trying to update state for an inexisting process", s, state)
		return
	}
	set[s] = state
}

func (set processSet) find(state ProcessState) *Process {
	for v, ps := range set {
		if ps&state > 0 {
			return v
		}
	}
	return nil
}

func (set processSet) all(state ProcessState) processSet {
	return set.choose(func(s *Process, state_ ProcessState) bool {
		return (state_&state > 0)
	})
}

func (set processSet) choose(fn func(*Process, ProcessState) bool) processSet {
	set2 := make(processSet)

	for s, ps := range set {
		if fn(s, ps) {
			set2[s] = ps
		}
	}

	return set2
}

func (set processSet) each(fn func(*Process)) {
	for s := range set {
		fn(s)
	}
}

func (set processSet) toSlice() []*Process {
	ary := make([]*Process, len(set))
	i := 0
	for v := range set {
		ary[i] = v
		i += 1
	}
	return ary
}

func (set processSet) starting() *Process {
	return set.find(PROCESS_STARTING)
}

func (set processSet) ready() *Process {
	return set.find(PROCESS_READY)
}

func (set processSet) stopping() []*Process {
	return set.all(PROCESS_STOPPING).toSlice()
}

// Utils

var EmptyProcessSet = make(processSet)
