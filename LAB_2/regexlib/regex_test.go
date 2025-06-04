// file: regexlib/regex_test.go
package regexlib

import (
	"strings"
	"testing"
)

// ------------------------------------------------------------------- helpers

func acc(t *testing.T, re *Regex, in string, want bool) {
	got := re.FindAll(in)
	if (len(got) > 0) != want {
		t.Fatalf("pattern %q on %q want %v got %v", re.pattern, in, want, got)
	}
}

func newRE(t *testing.T, pat string) *Regex {
	re, err := Compile(pat)
	if err != nil {
		t.Fatalf("compile %q: %v", pat, err)
	}
	return re
}

// ------------------------------------------------------------------- Lexer

func TestLexerTokens(t *testing.T) {
	l := newLexer(`a\*#|()[d-f]{3}\1`)
	want := []tokenType{
		tChar, tChar, tEpsilon, tUnion, tLParen, tRParen,
		tLBracket, tChar, tDash, tChar, tRBracket,
		tLBrace, tChar, tRBrace, tBackRef, tEOF,
	}
	for i, typ := range want {
		if tok := l.next(); tok.typ != typ {
			t.Fatalf("tok %d want %v got %v", i, typ, tok.typ)
		}
	}
}

// ------------------------------------------------------------------- Parser

func TestParserPrecedence(t *testing.T) {
	re := newRE(t, "a|bc*")
	acc(t, re, "a", true)
	acc(t, re, "bc", true)
	acc(t, re, "bccc", true)
	acc(t, re, "ab", false)
}

func TestParserCharClass(t *testing.T) {

	re := newRE(t, "[a-c]+")
	acc(t, re, "abcabc", true)
	acc(t, re, "d", false)

        re := newRE(t, "[a-c]+")
        acc(t, re, "abcabc", true)
        acc(t, re, "d", false)
}

func TestFindAllCharClass(t *testing.T) {
        re := newRE(t, "[a-c]")
        text := "zabcx"
        m := re.FindAll(text)
        if len(m) != 3 || text[m[0].Start:m[0].End] != "a" || text[m[1].Start:m[1].End] != "b" || text[m[2].Start:m[2].End] != "c" {
                t.Fatalf("unexpected matches %v", m)
        }
}

// ------------------------------------------------------------------- NFA ←→ DFA

// ------------------------------------------------------------------- Minimize

func TestMinimizeCount(t *testing.T) {
	re := newRE(t, "a|ab")
	before := len(re.dfa.States)
	min := Minimize(re.dfa)
	after := len(min.States)
	if before == after {
		t.Fatalf("expected fewer states (%d→?)", before)
	}
	if after != 2 {
		t.Fatalf("want 2 states got %d", after)
	}
}

// ------------------------------------------------------------------- FindAll + groups

func TestFindAllGroups(t *testing.T) {
	re := newRE(t, `(ab)c\1`)
	text := "xxabcab yy"
	m := re.FindAll(text)
	if len(m) != 1 || text[m[0].Start:m[0].End] != "abcab" {
		t.Fatalf("groups match wrong %v", m)
	}
}

// ------------------------------------------------------------------- Set-ops

func TestSetOps(t *testing.T) {
	a := newRE(t, "[ab]*")
	b := newRE(t, "a+")
	inter := a.Intersect(b)
	acc(t, inter, "aaa", true)
	acc(t, inter, "b", false)

	comp := a.Complement()
	acc(t, comp, "ccc", true)
	acc(t, comp, "aba", false)

	rev := newRE(t, "ab*").Reverse()
	acc(t, rev, "baa", true)
	acc(t, rev, "ab", false)
}

// ------------------------------------------------------------------- ToRegexp

func TestToRegexpPreservesLanguage(t *testing.T) {
	re := newRE(t, "a(b|c)*d")
	restored := newRE(t, re.ToRegexp())

	for _, s := range []string{"ad", "abcd", "abcbcd", "acbd"} {
		want := len(re.FindAll(s)) > 0
		got := len(restored.FindAll(s)) > 0
		if want != got {
			t.Fatalf("restore diff on %q", s)
		}
	}
}

// ------------------------------------------------------------------- Bench (quick)

func BenchmarkMillionAs(b *testing.B) {
	re := MustCompile("ab*")
	txt := strings.Repeat("a", 1_000_000)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = re.FindAll(txt)
	}
}

func TestNFAtoDFAEquivalence(t *testing.T) {
	pat := "(ab|a)*c"
	re := newRE(t, pat)

	// все строки длиной ≤4 из {a,b,c}
	alpha := []string{"", "a", "b", "c"}
	var words []string
	for _, x := range alpha {
		for _, y := range alpha {
			for _, z := range alpha {
				for _, w := range alpha {
					words = append(words, x+y+z+w)
				}
			}
		}
	}
	for _, s := range words {
		n := re.FindAll(s)
		d := re.ToRegexp() // ← используем публичный метод
		re2 := newRE(t, d)
		m := re2.FindAll(s)
		if (len(n) > 0) != (len(m) > 0) {
			t.Fatalf("equivalence fail on %q: %v vs %v", s, n, m)
		}
	}
}

// ------------------------------------------------------------------- end
