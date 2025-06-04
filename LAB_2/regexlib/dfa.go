package regexlib

import (
	"container/list"
	"fmt"
	"sort"
)

type dfaState struct {
	id     int
	accept bool
	trans  map[rune]*dfaState
	// group capture ids not tracked in DFA (simplified)
}

type DFA struct {
	Start  *dfaState
	States []*dfaState
	Alpha  []rune
}

func epsilonClosure(set map[*nfaState]struct{}) map[*nfaState]struct{} {
	stack := list.New()
	for s := range set {
		stack.PushBack(s)
	}
	for stack.Len() > 0 {
		elem := stack.Remove(stack.Back()).(*nfaState)
		for _, e := range elem.edges {
			if e.symbol == 0 {
				if _, ok := set[e.to]; !ok {
					set[e.to] = struct{}{}
					stack.PushBack(e.to)
				}
			}
		}
	}
	return set
}

func moveNFA(set map[*nfaState]struct{}, sym rune, runeset []rune) map[*nfaState]struct{} {
	res := make(map[*nfaState]struct{})
	for s := range set {
		for _, e := range s.edges {
			if e.symbol > 0 && e.symbol == sym {
				res[e.to] = struct{}{}
			} else if e.symbol == -1 {
				// char class
				for _, r := range e.set {
					if r == sym {
						res[e.to] = struct{}{}
						break
					}
				}
			}
		}
	}
	return res
}

func nfaToDFAcore(start *nfaState, alpha []rune) *DFA {
	// initial
	initSet := epsilonClosure(map[*nfaState]struct{}{start: {}})
	key := func(set map[*nfaState]struct{}) string {
		ids := make([]int, 0, len(set))
		for s := range set {
			ids = append(ids, s.id)
		}
		sort.Ints(ids)
		return fmt.Sprint(ids)
	}
	mp := map[string]*dfaState{}
	dStart := &dfaState{id: 0, trans: map[rune]*dfaState{}}
	mp[key(initSet)] = dStart
	if hasAccept(initSet) {
		dStart.accept = true
	}
	queue := []map[*nfaState]struct{}{initSet}
	states := []*dfaState{dStart}
	for len(queue) > 0 {
		curSet := queue[0]
		queue = queue[1:]
		curKey := key(curSet)
		curD := mp[curKey]
		for _, sym := range alpha {
			moveSet := moveNFA(curSet, sym, alpha)
			if len(moveSet) == 0 {
				continue
			}
			clo := epsilonClosure(moveSet)
			k := key(clo)
			d, exists := mp[k]
			if !exists {
				d = &dfaState{id: len(states), trans: map[rune]*dfaState{}}
				if hasAccept(clo) {
					d.accept = true
				}
				mp[k] = d
				states = append(states, d)
				queue = append(queue, clo)
			}
			curD.trans[sym] = d
		}
	}
	return &DFA{Start: dStart, States: states, Alpha: alpha}
}

func hasAccept(set map[*nfaState]struct{}) bool {
	for s := range set {
		if s.accept {
			return true
		}
	}
	return false
}
