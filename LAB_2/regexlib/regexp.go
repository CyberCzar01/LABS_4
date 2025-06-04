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
	pos := 0
	curr := epsilonClosure(map[*nfaState]struct{}{r.nfaStart: {}})

	// отметим группы, открывающиеся из ε-переходов старта
	for st := range curr {
		for _, g := range st.openGroups {
			starts[g] = pos
		}
	}

	for pos < len(s) {
		ch, sz := utf8.DecodeRuneInString(s[pos:])
		next := map[*nfaState]struct{}{}

                for st := range curr {
                        for _, e := range st.edges {
                                switch {
                                case e.symbol == ch:
                                        next[e.to] = struct{}{}
                                case e.symbol == -1:
                                        for _, r := range e.set {
                                                if r == ch {
                                                        next[e.to] = struct{}{}
                                                        break
                                                }
                                        }
                                }
                        }
                }

		next = epsilonClosure(next)
		if len(next) == 0 {
			break
		}

		pos += sz
		curr = next

		for st := range curr {
			for _, g := range st.openGroups {
				// --- исправленное смещение ---
				starts[g] = pos
			}
			for _, g := range st.closeGroups {
				ends[g] = pos
			}
		}
	}

	for st := range curr {
		if st.accept {
			return pos
		}
	}
	return 0
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
