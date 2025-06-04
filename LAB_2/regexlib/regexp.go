// file: regexlib/regex.go
package regexlib

import (
	"errors"
	"unicode/utf8"
)

/* ----------- Компиляция ----------- */

type Regex struct {
	pattern   string
	ast       *astNode
	numGroups int

	nfaStart  *nfaState
	nfaAccept *nfaState
	rawDFA    *DFA
	dfa       *DFA
	alphabet  []rune
}

func Compile(pattern string) (*Regex, error) {
	if pattern == "" {
		return nil, errors.New("empty pattern")
	}

	/* 1) парсинг ---------------------------------------------------------- */
	p := newParser(pattern)
	ast, err := p.parse()
	if err != nil {
		return nil, err
	}

	/* 2) Thompson-NFA ----------------------------------------------------- */
	startNFA, acceptNFA := compileASTtoNFA(ast)

	/* 3) алфавит ---------------------------------------------------------- */
	alphaSet := map[rune]struct{}{}
	var walk func(*astNode)
	walk = func(n *astNode) {
		if n == nil {
			return
		}
		switch n.typ {
		case nChar:
			if n.ch != 0 { // ε-переходы не добавляем
				alphaSet[n.ch] = struct{}{}
			}
		case nSet: // если у вас есть классы символов
			for _, r := range n.charset {
				alphaSet[r] = struct{}{}
			}
		}
		walk(n.left)
		walk(n.right)
	}
	walk(ast)

	alphabet := make([]rune, 0, len(alphaSet))
	for r := range alphaSet {
		alphabet = append(alphabet, r)
	}

	/* 4) NFA → DFA -------------------------------------------------------- */
	raw := nfaToDFAcore(startNFA, alphabet)
	raw.Alpha = alphabet

	/* 5) минимизация ------------------------------------------------------ */
	min := Minimize(raw)
	// переносим алфавит — важно для визуализации и set-операций
	min.Alpha = alphabet

	return &Regex{
		pattern:   pattern,
		ast:       ast,
		numGroups: countGroups(ast),
		nfaStart:  startNFA,
		nfaAccept: acceptNFA,
		rawDFA:    raw,
		dfa:       min,
		alphabet:  alphabet,
	}, nil
}

func MustCompile(p string) *Regex {
	r, err := Compile(p)
	if err != nil {
		panic(err)
	}
	return r
}

/* ----------- Поиск с группами ----------------------------------------- */

type Match struct {
	Start, End int
	Groups     []string
}

// FindSubmatchAt ищет ближайшее совпадение, начиная с pos.
func (r *Regex) FindSubmatchAt(text string, pos int) ([]string, int) {
	// Если NFA недоступен (Regex получен через операции над DFA),
	// используем простую симуляцию DFA без поддержки групп.
	if r.nfaStart == nil {
		l := r.matchDFA(text[pos:])
		if l == 0 {
			return nil, 0
		}
		return []string{text[pos : pos+l]}, l
	}

	G := r.numGroups
	starts := make([]int, G+1)
	ends := make([]int, G+1)

	length := r.matchWithGroups(text[pos:], starts, ends)
	if length == 0 {
		return nil, 0
	}

	subs := make([]string, G+1)
	subs[0] = text[pos : pos+length]
	for i := 1; i <= G; i++ {
		if ends[i] >= starts[i] && ends[i] <= len(text[pos:]) {
			subs[i] = text[pos+starts[i] : pos+ends[i]]
		}
	}
	return subs, length
}

// FindAll возвращает все непересекающиеся матчи.
func (r *Regex) FindAll(text string) []Match {
	var out []Match
	for i := 0; i < len(text); {
		subs, l := r.FindSubmatchAt(text, i)
		if l == 0 {
			// шаг вперёд на одну руну
			_, sz := utf8.DecodeRuneInString(text[i:])
			i += sz
			continue
		}
		out = append(out, Match{
			Start:  i,
			End:    i + l,
			Groups: subs[1:], // без «whole match»
		})
		i += l
	}
	return out
}

/* ------------------- внутренняя симуляция NFA ------------------------- */

func (r *Regex) matchWithGroups(s string, starts, ends []int) int {
	runes := []rune(s)

	type item struct {
		st     *nfaState
		pos    int // index in runes
		starts []int
		ends   []int
	}

	enqueue := func(q *[]item, it item) {
		*q = append(*q, it)
	}

	// initial closure
	initStarts := append([]int(nil), starts...)
	initEnds := append([]int(nil), ends...)
	startItems := []item{{st: r.nfaStart, pos: 0, starts: initStarts, ends: initEnds}}
	queue := startItems
	visited := make(map[[2]int]bool) // state id + pos

	bestLen := 0
	var bestStarts, bestEnds []int

	for len(queue) > 0 {
		it := queue[0]
		queue = queue[1:]

		if visited[[2]int{it.st.id, it.pos}] {
			continue
		}
		visited[[2]int{it.st.id, it.pos}] = true

		// apply group markers when entering state
		for _, g := range it.st.openGroups {
			it.starts[g] = it.pos
		}
		for _, g := range it.st.closeGroups {
			it.ends[g] = it.pos
		}

		if it.st.accept {
			if it.pos > bestLen {
				bestLen = it.pos
				bestStarts = append([]int(nil), it.starts...)
				bestEnds = append([]int(nil), it.ends...)
			}
		}

		for _, e := range it.st.edges {
			switch {
			case e.symbol == 0:
				enqueue(&queue, item{st: e.to, pos: it.pos, starts: append([]int(nil), it.starts...), ends: append([]int(nil), it.ends...)})
			case e.symbol == -1:
				if it.pos < len(runes) {
					ch := runes[it.pos]
					for _, r := range e.set {
						if r == ch {
							enqueue(&queue, item{st: e.to, pos: it.pos + 1, starts: append([]int(nil), it.starts...), ends: append([]int(nil), it.ends...)})
							break
						}
					}
				}
			case e.symbol == -2:
				grp := int(e.set[0])
				sub := runes[it.starts[grp]:it.ends[grp]]
				if len(sub) == 0 {
					enqueue(&queue, item{st: e.to, pos: it.pos, starts: append([]int(nil), it.starts...), ends: append([]int(nil), it.ends...)})
				} else if it.pos+len(sub) <= len(runes) {
					if equalRuneSlice(runes[it.pos:it.pos+len(sub)], sub) {
						enqueue(&queue, item{st: e.to, pos: it.pos + len(sub), starts: append([]int(nil), it.starts...), ends: append([]int(nil), it.ends...)})
					}
				}
			default:
				if it.pos < len(runes) && runes[it.pos] == e.symbol {
					enqueue(&queue, item{st: e.to, pos: it.pos + 1, starts: append([]int(nil), it.starts...), ends: append([]int(nil), it.ends...)})
				}
			}
		}
	}

	if bestLen > 0 {
		copy(starts, bestStarts)
		copy(ends, bestEnds)
	}

	// convert rune index to byte length
	return len(string(runes[:bestLen]))
}

/* ----------- Сервисные геттеры --------------------------------------- */

func (r *Regex) DFA() *DFA      { return r.dfa }
func (r *Regex) RawDFA() *DFA   { return r.rawDFA }
func (r *Regex) NFA() *nfaState { return r.nfaStart }

/* ----------- Вспомогательные ----------------------------------------- */

func countGroups(n *astNode) int {
	if n == nil {
		return 0
	}
	max := 0
	if n.typ == nGroup && n.grpNum > max {
		max = n.grpNum
	}
	if m := countGroups(n.left); m > max {
		max = m
	}
	if m := countGroups(n.right); m > max {
		max = m
	}
	return max
}

// -------------------- вспомогательные для DFA-базовых Regex ------------

// regexFromDFA создаёт Regex только по автомату. Группы и AST отсутствуют,
// поэтому поддерживается лишь поиск совпадений без захватывающих групп.
func regexFromDFA(d *DFA) *Regex {
	min := Minimize(d)
	min.Alpha = d.Alpha
	return &Regex{dfa: min, rawDFA: d, alphabet: min.Alpha}
}

func equalRuneSlice(a, b []rune) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// matchDFA возвращает длину наибольшего префикса строки, принимаемого DFA.
func (r *Regex) matchDFA(s string) int {
	if r.dfa == nil {
		return 0
	}
	state := r.dfa.Start
	lastAccept := -1
	pos := 0
	for pos < len(s) {
		ch, sz := utf8.DecodeRuneInString(s[pos:])
		next, ok := state.trans[ch]
		if !ok {
			break
		}
		pos += sz
		state = next
		if state.accept {
			lastAccept = pos
		}
	}
	if lastAccept >= 0 {
		return lastAccept
	}
	return 0
}

// Complement возвращает Regex, распознающий дополнение языка исходного DFA.
func (r *Regex) Complement() *Regex {
	d := cloneDFA(r.dfa)
	d.Alpha = unionRunes(d.Alpha, asciiAlphabet())
	d = completeDFA(d)
	for _, s := range d.States {
		s.accept = !s.accept
	}
	return regexFromDFA(d)
}

// Intersect возвращает Regex, распознающий пересечение языков двух DFA.
func (r *Regex) Intersect(o *Regex) *Regex {
	d := IntersectDFA(r.dfa, o.dfa)
	return regexFromDFA(d)
}

// Reverse строит Regex, распознающий развёрнутый язык исходного DFA.
func (r *Regex) Reverse() *Regex {
	revAST := reverseAST(r.ast)
	start, _ := compileASTtoNFA(revAST)
	dfa := nfaToDFAcore(start, r.alphabet)
	dfa.Alpha = r.alphabet
	return regexFromDFA(dfa)
}

func reverseAST(n *astNode) *astNode {
	if n == nil {
		return nil
	}
	switch n.typ {
	case nConcat:
		return &astNode{typ: nConcat, left: reverseAST(n.right), right: reverseAST(n.left)}
	case nUnion:
		return &astNode{typ: nUnion, left: reverseAST(n.left), right: reverseAST(n.right)}
	case nStar, nPlus, nQMark:
		return &astNode{typ: n.typ, left: reverseAST(n.left)}
	case nRepeat:
		return &astNode{typ: nRepeat, left: reverseAST(n.left), min: n.min, max: n.max}
	case nGroup:
		return &astNode{typ: nGroup, left: reverseAST(n.left), grpNum: n.grpNum}
	default:
		cp := *n
		cp.left = nil
		cp.right = nil
		return &cp
	}
}

// ToRegexp восстанавливает регулярное выражение из минимального DFA.
func (r *Regex) ToRegexp() string {
	if r.dfa == nil {
		return ""
	}
	return r.dfa.ToRegexp()
}
