package crank

type supervisorSet map[*Supervisor]bool

func (set supervisorSet) add(s *Supervisor) {
	set[s] = true
}

func (set supervisorSet) rem(s *Supervisor) {
	delete(set, s)
}

func (set supervisorSet) len() int {
	return len(set)
}

func (set supervisorSet) find(state *ProcessState) *Supervisor {
	for v := range set {
		if v.state == state {
			return v
		}
	}
	return nil
}

func (set supervisorSet) all(state *ProcessState) supervisorSet {
	return set.choose(func(s *Supervisor) bool {
		return s.state == state
	})
}

func (set supervisorSet) choose(fn func(*Supervisor) bool) supervisorSet {
	set2 := make(supervisorSet)

	for v := range set {
		if fn(v) {
			set2[v] = true
		}
	}

	return set2
}

func (set supervisorSet) each(fn func(*Supervisor)) {
	for v := range set {
		fn(v)
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