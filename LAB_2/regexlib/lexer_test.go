package regexlib

import "testing"

func TestLexer(t *testing.T) {
	l := newLexer("a{3}b|c#\\1[d-f]")
	tok := l.next()
	if tok.typ != tChar || tok.ch != 'a' {
		t.Fail()
	}
}
