package regexlib

import "sort"

// complement on complete DFA (assumes total transition function)
func Complement(d *DFA) *DFA {
	d = cloneDFA(d)
	d = completeDFA(d)
	for _, s := range d.States {
		s.accept = !s.accept
	}
	return d
}

func Product(a, b *DFA, op func(bool, bool) bool) *DFA {
	type pair struct{ i, j int }
	mp := map[pair]*dfaState{}
	startPair := pair{a.Start.id, b.Start.id}
	start := &dfaState{id: 0, accept: op(a.Start.accept, b.Start.accept), trans: map[rune]*dfaState{}}
	mp[startPair] = start
	queue := []pair{startPair}
	states := []*dfaState{start}
	alpha := unionRunes(a.Alpha, b.Alpha)
	for len(queue) > 0 {
		p := queue[0]
		queue = queue[1:]
		cur := mp[p]
		for _, c := range alpha {
			ta, oka := a.States[p.i].trans[c]
			tb, okb := b.States[p.j].trans[c]
			if !oka || !okb {
				continue
			}
			np := pair{ta.id, tb.id}
			ns, exists := mp[np]
			if !exists {
				ns = &dfaState{id: len(states), accept: op(ta.accept, tb.accept), trans: map[rune]*dfaState{}}
				mp[np] = ns
				states = append(states, ns)
				queue = append(queue, np)
			}
			cur.trans[c] = ns
		}
	}
	return &DFA{Start: start, States: states, Alpha: alpha}
}

func unionRunes(a, b []rune) []rune {
	m := map[rune]struct{}{}
	for _, r := range a {
		m[r] = struct{}{}
	}
	for _, r := range b {
		m[r] = struct{}{}
	}
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func asciiAlphabet() []rune {
	out := make([]rune, 128)
	for i := 0; i < 128; i++ {
		out[i] = rune(i)
	}
	return out
}

// Intersection: op = &&
func IntersectDFA(a, b *DFA) *DFA { return Product(a, b, func(x, y bool) bool { return x && y }) }

// Union: op = ||
func UnionDFA(a, b *DFA) *DFA { return Product(a, b, func(x, y bool) bool { return x || y }) }

// Reverse language: make NFA by reversing edges then determinise
func ReverseDFA(d *DFA) *DFA {
	nodes := make([]*nfaState, len(d.States))
	for i := range nodes {
		nodes[i] = newState()
	}
	start := newState()
	for _, s := range d.States {
		if s.accept {
			start.edges = append(start.edges, &nfaEdge{symbol: 0, to: nodes[s.id]})
		}
	}
	for _, s := range d.States {
		for c, to := range s.trans {
			nodes[to.id].edges = append(nodes[to.id].edges, &nfaEdge{symbol: c, to: nodes[s.id]})
		}
	}
	nodes[d.Start.id].accept = true
	dfa := nfaToDFAcore(start, d.Alpha)
	dfa.Alpha = d.Alpha
	return Minimize(dfa)
}

// cloneDFA создает глубокую копию автомата.
func cloneDFA(d *DFA) *DFA {
	states := make([]*dfaState, len(d.States))
	for i, s := range d.States {
		states[i] = &dfaState{id: i, accept: s.accept, trans: map[rune]*dfaState{}}
	}
	for i, s := range d.States {
		for c, t := range s.trans {
			states[i].trans[c] = states[t.id]
		}
	}
	return &DFA{Start: states[d.Start.id], States: states, Alpha: append([]rune(nil), d.Alpha...)}
}

// completeDFA добавляет поглощающее состояние для отсутствующих переходов.
func completeDFA(d *DFA) *DFA {
	sink := &dfaState{id: len(d.States), trans: map[rune]*dfaState{}}
	for _, c := range d.Alpha {
		sink.trans[c] = sink
	}
	for _, s := range d.States {
		for _, c := range d.Alpha {
			if _, ok := s.trans[c]; !ok {
				s.trans[c] = sink
			}
		}
	}
	d.States = append(d.States, sink)
	return d
}
