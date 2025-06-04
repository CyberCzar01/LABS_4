package regexlib

import (
	"fmt"
	"io"
)

// ExportDOT печатает Graphviz-представление NFA или DFA в w.
func ExportDOT(w io.Writer, g interface{}) {
	fmt.Fprintln(w, "digraph G {")
	fmt.Fprintln(w, "    rankdir=LR;")

	switch t := g.(type) {

	//------------------------------------------------------------------ DFA
	case *DFA:
		for _, s := range t.States {
			shape := "circle"
			if s.accept {
				shape = "doublecircle"
			}
			fmt.Fprintf(w, "    q%d [shape=%s];\n", s.id, shape)
			for ch, to := range s.trans {
				fmt.Fprintf(w, "    q%d -> q%d [label=\"%c\"];\n", s.id, to.id, ch)
			}
		}
		fmt.Fprintf(w, "    _start [shape=point]; _start -> q%d;\n", t.Start.id)

	//------------------------------------------------------------------ NFA
	case *nfaState:
		visited := map[*nfaState]bool{}
		var dfs func(*nfaState)
		dfs = func(s *nfaState) {
			if visited[s] {
				return
			}
			visited[s] = true
			shape := "circle"
			if s.accept {
				shape = "doublecircle"
			}
			fmt.Fprintf(w, "    n%d [shape=%s];\n", s.id, shape)

			for _, e := range s.edges {
				label := "ε"
				switch {
				case e.symbol == 0:
					label = "ε"
				case e.symbol == -1:
					label = "class"
				case e.symbol == -2:
					label = fmt.Sprintf("\\%d", e.set[0])
				default:
					label = string(e.symbol)
				}
				fmt.Fprintf(w, "    n%d -> n%d [label=\"%s\"];\n", s.id, e.to.id, label)
				dfs(e.to)
			}
		}
		dfs(t)
		fmt.Fprintf(w, "    _start [shape=point]; _start -> n%d;\n", t.id)

	default:
		fmt.Fprintln(w, "    /* unknown graph type */")
	}

	fmt.Fprintln(w, "}")
}
