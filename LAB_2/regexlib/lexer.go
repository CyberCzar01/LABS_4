package regexlib

import (
	"unicode/utf8"
)

type tokenType int

const (
	tEOF      tokenType = iota
	tChar               // literal rune
	tLParen             // (
	tRParen             // )
	tStar               // *
	tPlus               // +
	tQMark              // ?
	tUnion              // |
	tLBracket           // [
	tRBracket           // ]
	tDash               // - inside []
	tLBrace             // {
	tRBrace             // }
	tComma              // , (for {m,n})
	tEpsilon            // #
	tBackRef            // \1..\9 etc
)

type token struct {
	typ tokenType
	ch  rune // for tChar
	num int  // for tBackRef, {n}
}

type lexer struct {
	input string
	pos   int
}

func newLexer(s string) *lexer { return &lexer{input: s} }

func (l *lexer) next() token {
	if l.pos >= len(l.input) {
		return token{typ: tEOF}
	}
	r, size := utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += size
	switch r {
	case '(':
		return token{typ: tLParen}
	case ')':
		return token{typ: tRParen}
	case '*':
		return token{typ: tStar}
	case '+':
		return token{typ: tPlus}
	case '?':
		return token{typ: tQMark}
	case '|':
		return token{typ: tUnion}
	case '[':
		return token{typ: tLBracket}
	case ']':
		return token{typ: tRBracket}
	case '-':
		return token{typ: tDash}
	case '{':
		return token{typ: tLBrace}
	case '}':
		return token{typ: tRBrace}
	case ',':
		return token{typ: tComma}
	case '#':
		return token{typ: tEpsilon}
	case '\\':
		if l.pos >= len(l.input) {
			// standalone backslash => treat as literal
			return token{typ: tChar, ch: r}
		}
		// lookahead
		r2, s2 := utf8.DecodeRuneInString(l.input[l.pos:])
		if r2 >= '0' && r2 <= '9' {
			l.pos += s2
			return token{typ: tBackRef, num: int(r2 - '0')}
		}
		l.pos += s2
		return token{typ: tChar, ch: r2}
	default:
		return token{typ: tChar, ch: r}
	}
}
