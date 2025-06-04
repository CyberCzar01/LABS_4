package regexlib

import "sort"

// complement on complete DFA (assumes total transition function)
func Complement(d *DFA) *DFA {
	// shallow copy states first
	newStates := make([]*dfaState, len(d.States))
	for i, s := range d.States {
		newStates[i] = &dfaState{id: i, accept: !s.accept, trans: map[rune]*dfaState{}}
	}
	for i, s := range d.States {
		for c, t := range s.trans {
			newStates[i].trans[c] = newStates[t.id]
		}
	}
	return &DFA{Start: newStates[d.Start.id], States: newStates, Alpha: d.Alpha}
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

// Intersection: op = &&
func IntersectDFA(a, b *DFA) *DFA { return Product(a, b, func(x, y bool) bool { return x && y }) }

// Union: op = ||
func UnionDFA(a, b *DFA) *DFA { return Product(a, b, func(x, y bool) bool { return x || y }) }

// Reverse language: make NFA by reversing edges then determinise
func ReverseDFA(d *DFA) *DFA {
	// build reverse NFA
	nodes := make([]*nfaState, len(d.States))
	for i := range nodes {
		nodes[i] = newState()
	}
	start := newState()
	acceptSet := map[*nfaState]struct{}{}
	for _, s := range d.States {
		if s.accept {
			acceptSet[nodes[s.id]] = struct{}{}
		}
	}
	// ε from new start to each accept of original
	for acc := range acceptSet {
		start.edges = append(start.edges, &nfaEdge{symbol: 0, to: acc})
	}
	// reverse transitions
	for _, s := range d.States {
		for c, to := range s.trans {
			nodes[to.id].edges = append(nodes[to.id].edges, &nfaEdge{symbol: c, to: nodes[s.id]})
		}
	}
	acceptDummy := newState()
	acceptDummy.accept = true
	return nfaToDFA(start, acceptDummy, d.Alpha)
}
