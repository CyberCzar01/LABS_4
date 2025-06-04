package regexlib

import "strings"

// ToRegexp превращает минимальный DFA в эквивалентное регулярное выражение
// методом исключения состояний (McNaughton-Yamada).
func (d *DFA) ToRegexp() string {
	if d == nil || len(d.States) == 0 {
		return "∅"
	}

	n := len(d.States)
	R := make([][]string, n)
	for i := range R {
		R[i] = make([]string, n)
	}

	// 1. инициализируем прямые рёбра
	for _, s := range d.States {
		for c, t := range s.trans {
			lex := escapeRune(c)
			if R[s.id][t.id] == "" {
				R[s.id][t.id] = lex
			} else {
				R[s.id][t.id] += "|" + lex
			}
		}
	}

	start := d.Start.id
	finals := make([]int, 0, len(d.States))
	for _, s := range d.States {
		if s.accept {
			finals = append(finals, s.id)
		}
	}

	// 2. последовательно исключаем промежуточные состояния
	for k := 0; k < n; k++ {
		for i := 0; i < n; i++ {
			if i == k {
				continue
			}
			for j := 0; j < n; j++ {
				if j == k {
					continue
				}

				rik := R[i][k]
				rkk := R[k][k]
				rkj := R[k][j]

				if rik == "" || rkj == "" {
					continue
				}

				var middle string
				if rkk != "" {
					middle = "(" + rkk + ")*"
				}
				expr := concat(regexAlt(rik), middle, regexAlt(rkj))

				if R[i][j] == "" {
					R[i][j] = expr
				} else {
					R[i][j] += "|" + expr
				}
			}
		}
	}

	// 3. собираем выражение между стартом и любым финалом
	var resultParts []string
	for _, f := range finals {
		if part := R[start][f]; part != "" {
			resultParts = append(resultParts, part)
		}
	}
	if len(resultParts) == 0 {
		return "∅"
	}
	return strings.Join(resultParts, "|")
}

// --- вспомогательные функции ------------------------------------------------

func escapeRune(r rune) string {
	switch r {
	case '*', '+', '?', '|', '(', ')', '[', ']', '{', '}', '.':
		return "\\" + string(r)
	default:
		return string(r)
	}
}

func concat(parts ...string) string {
	var b strings.Builder
	for _, p := range parts {
		if p != "" {
			b.WriteString(p)
		}
	}
	return b.String()
}

func regexAlt(s string) string {
	if strings.ContainsRune(s, '|') {
		return "(" + s + ")"
	}
	return s
}
