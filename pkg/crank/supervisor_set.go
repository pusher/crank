package crank

type supervisorSet map[*Supervisor]*ProcessState

func (set supervisorSet) add(s *Supervisor) {
	set[s] = PROCESS_NEW
}

func (set supervisorSet) rem(s *Supervisor) {
	delete(set, s)
}

func (set supervisorSet) len() int {
	return len(set)
}

func (set supervisorSet) updateState(s *Supervisor, state *ProcessState) {
	if _, ok := set[s]; !ok {
		fail("Trying to update state for an inexisting supervisor", s, state)
		return
	}
	set[s] = state
}

func (set supervisorSet) find(state *ProcessState) *Supervisor {
	for v, ps := range set {
		if ps == state {
			return v
		}
	}
	return nil
}

func (set supervisorSet) all(state *ProcessState) supervisorSet {
	return set.choose(func(s *Supervisor, state_ *ProcessState) bool {
		return state_ == state
	})
}

func (set supervisorSet) choose(fn func(*Supervisor, *ProcessState) bool) supervisorSet {
	set2 := make(supervisorSet)

	for s, ps := range set {
		if fn(s, ps) {
			set2[s] = ps
		}
	}

	return set2
}

func (set supervisorSet) each(fn func(*Supervisor)) {
	for s := range set {
		fn(s)
	}
}

func (set supervisorSet) toSlice() []*Supervisor {
	ary := make([]*Supervisor, len(set))
	i := 0
	for v := range set {
		ary[i] = v
		i += 1
	}
	return ary
}

func (set supervisorSet) starting() *Supervisor {
	s := set.find(PROCESS_STARTING)
	if s == nil {
		s = set.find(PROCESS_NEW)
	}
	return s
}

func (set supervisorSet) current() *Supervisor {
	return set.find(PROCESS_READY)
}

func (set supervisorSet) stopping() []*Supervisor {
	return set.all(PROCESS_STOPPING).toSlice()
}

// Utils

var EmptySupervisorSet = make(supervisorSet)
