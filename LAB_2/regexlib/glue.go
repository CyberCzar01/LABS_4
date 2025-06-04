package regexlib

// ---------------------------------------------------------------------------
// Совместимость: compileASTtoNFA (ожидает regexp.go)
// ---------------------------------------------------------------------------

func compileASTtoNFA(root *astNode) (start, accept *nfaState) {
	frag := buildNFA(root)

	accept = newState()
	accept.accept = true
	patchOuts(frag.outs, accept)

	return frag.start, accept
}

// ---------------------------------------------------------------------------
// Совместимость: nfaToDFA(start, accept, alphabet)
// ---------------------------------------------------------------------------

// Старая сигнатура с тремя параметрами вызывает «ядро» из dfa.go.
func nfaToDFA(start, accept *nfaState, alphabet []rune) *DFA {
	return nfaToDFAcore(start, alphabet)
}
