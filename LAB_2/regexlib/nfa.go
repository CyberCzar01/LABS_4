package regexlib

var stateID int

func nextStateID() int { stateID++; return stateID - 1 }

type nfaState struct {
	id     int
	edges  []*nfaEdge
	accept bool
	// group markers
	openGroups  []int
	closeGroups []int
}

type nfaEdge struct {
	symbol rune // 0 = ε, -1 indicates char class, -2 backref placeholder
	set    []rune
	to     *nfaState
}

type nfaFrag struct {
	start *nfaState
	outs  []*nfaState // dangling ε edges that need patching to new state
}

func newState() *nfaState { return &nfaState{id: nextStateID()} }

func patchOuts(outs []*nfaState, to *nfaState) {
	for _, s := range outs {
		s.edges = append(s.edges, &nfaEdge{symbol: 0, to: to})
	}
}

func buildNFA(node *astNode) nfaFrag {
	switch node.typ {
	case nEmpty:
		s := newState()
		return nfaFrag{start: s, outs: []*nfaState{s}}
	case nChar:
		s1 := newState()
		s2 := newState()
		s1.edges = append(s1.edges, &nfaEdge{symbol: node.ch, to: s2})
		return nfaFrag{start: s1, outs: []*nfaState{s2}}
	case nSet:
		s1 := newState()
		s2 := newState()
		s1.edges = append(s1.edges, &nfaEdge{symbol: -1, set: node.charset, to: s2})
		return nfaFrag{start: s1, outs: []*nfaState{s2}}
	case nConcat:
		f1 := buildNFA(node.left)
		f2 := buildNFA(node.right)
		patchOuts(f1.outs, f2.start)
		return nfaFrag{start: f1.start, outs: f2.outs}
	case nUnion:
		s := newState()
		f1 := buildNFA(node.left)
		f2 := buildNFA(node.right)
		s.edges = append(s.edges, &nfaEdge{symbol: 0, to: f1.start})
		s.edges = append(s.edges, &nfaEdge{symbol: 0, to: f2.start})
		outs := append(f1.outs, f2.outs...)
		return nfaFrag{start: s, outs: outs}
	case nStar:
		s := newState()
		f := buildNFA(node.left)
		patchOuts(f.outs, s)
		s.edges = append(s.edges, &nfaEdge{symbol: 0, to: f.start})
		return nfaFrag{start: s, outs: []*nfaState{s}}
	case nPlus:
		f := buildNFA(node.left)
		patchOuts(f.outs, f.start)
		return f
	case nQMark:
		s := newState()
		f := buildNFA(node.left)
		s.edges = append(s.edges, &nfaEdge{symbol: 0, to: f.start})
		outs := append(f.outs, s)
		return nfaFrag{start: s, outs: outs}
	case nRepeat:
		if node.max != -1 && node.max < node.min {
			panic("repeat max<min")
		}
		// naive expansion
		var frag nfaFrag
		for i := 0; i < node.min; i++ {
			piece := buildNFA(node.left)
			if i == 0 {
				frag = piece
			} else {
				patchOuts(frag.outs, piece.start)
				frag.outs = piece.outs
			}
		}
		optionalCount := node.max - node.min
		if node.max == -1 {
			optionalCount = 2
		} // treat {n,} as {n,∞} approx with star
		for i := 0; i < optionalCount; i++ {
			// each optional piece is left? (qmark)
			piece := buildNFA(node.left)
			// create ε branch to skip piece
			skipState := newState()
			patchOuts(frag.outs, skipState)
			patchOuts(frag.outs, piece.start)
			frag.outs = append(piece.outs, skipState)
		}
		return frag
	case nGroup:
		frag := buildNFA(node.left)
		frag.start.openGroups = append(frag.start.openGroups, node.grpNum)
		for _, o := range frag.outs {
			o.closeGroups = append(o.closeGroups, node.grpNum)
		}
		return frag
	case nBackRef:
		s1 := newState()
		s2 := newState()
		s1.edges = append(s1.edges, &nfaEdge{symbol: -2, to: s2, set: []rune{rune(node.grpNum)}})
		return nfaFrag{start: s1, outs: []*nfaState{s2}}
	default:
		panic("unknown ast node")
	}
}
