package regexlib

import (
	"fmt"
	"strconv"
)

type parser struct {
	lex       *lexer
	look      token
	nextGroup int
}

func newParser(pat string) *parser {
	p := &parser{lex: newLexer(pat), nextGroup: 1}
	p.look = p.lex.next()
	return p
}

func (p *parser) scan() { p.look = p.lex.next() }

// Pratt-парсер
func (p *parser) parse() (*astNode, error) { return p.parseExpr(1) }

func precedence(t tokenType) int {
	switch t {
	case tUnion:
		return 1
	case tChar, tLParen, tLBracket, tEpsilon, tBackRef:
		return 2 // неявная конкатенация
	case tStar, tPlus, tQMark, tLBrace:
		return 3
	default:
		return 0
	}
}

func (p *parser) parseExpr(minPrec int) (*astNode, error) {
	// ---------- префикс ----------
	var left *astNode
	switch p.look.typ {
	case tChar:
		left = charNode(p.look.ch)
		p.scan()
	case tEpsilon:
		left = &astNode{typ: nEmpty}
		p.scan()
	case tLParen:
		p.scan()
		inner, err := p.parseExpr(1)
		if err != nil {
			return nil, err
		}
		if p.look.typ != tRParen {
			return nil, fmt.Errorf("expected )")
		}
		left = &astNode{typ: nGroup, left: inner, grpNum: p.nextGroup}
		p.nextGroup++
		p.scan()
	case tLBracket:
		p.scan() // поглощаем '['
		set, err := p.parseCharClass()
		if err != nil {
			return nil, err
		}
		left = &astNode{typ: nSet, charset: set}
	case tBackRef:
		left = &astNode{typ: nBackRef, grpNum: p.look.num}
		p.scan()
	default:
		return nil, fmt.Errorf("unexpected token %v", p.look.typ)
	}

	// ---------- суффиксы (* + ? {m,n}) ----------
	for {
		switch p.look.typ {
		case tStar:
			left = &astNode{typ: nStar, left: left}
			p.scan()
		case tPlus:
			left = &astNode{typ: nPlus, left: left}
			p.scan()
		case tQMark:
			left = &astNode{typ: nQMark, left: left}
			p.scan()
		case tLBrace:
			min, max, err := p.parseRepeat()
			if err != nil {
				return nil, err
			}
			left = &astNode{typ: nRepeat, left: left, min: min, max: max}
		default:
			goto noPostfix
		}
	}
noPostfix:

	// ---------- инфиксы (конкатенация, |) ----------
	for precedence(p.look.typ) >= minPrec {
		tok := p.look
		var prec int
		switch tok.typ {
		case tUnion:
			prec = 1
			p.scan() // съедаем '|'
		default: // неявная конкатенация
			prec = 2
			// НЕ двигаем p.scan(): текущий токен уже начало RHS
			tok.typ = 999 // фиктивный
		}

		nextMin := prec + 1 // левая ассоц. для обоих операторов
		right, err := p.parseExpr(nextMin)
		if err != nil {
			return nil, err
		}

		if tok.typ == tUnion {
			left = &astNode{typ: nUnion, left: left, right: right}
		} else {
			left = &astNode{typ: nConcat, left: left, right: right}
		}
	}

	return left, nil
}

/* ----------------------- вспомогательные парсеры ----------------------- */

func (p *parser) parseCharClass() ([]rune, error) {
	negate := false
	set := map[rune]struct{}{}

	if p.look.typ == tChar && p.look.ch == '^' {
		negate = true
		p.scan()
	}

	for p.look.typ != tRBracket && p.look.typ != tEOF {
		if p.look.typ != tChar {
			return nil, fmt.Errorf("invalid char class token")
		}
		start := p.look.ch
		p.scan()

		if p.look.typ == tDash {
			p.scan()
			if p.look.typ != tChar {
				return nil, fmt.Errorf("incomplete range")
			}
			end := p.look.ch
			p.scan()
			for r := start; r <= end; r++ {
				set[r] = struct{}{}
			}
		} else {
			set[start] = struct{}{}
		}
	}

	if p.look.typ != tRBracket {
		return nil, fmt.Errorf("missing ]")
	}
	p.scan() // потребляем ']'

	out := make([]rune, 0, len(set))
	for r := range set {
		out = append(out, r)
	}
	if negate {
		full := make([]rune, 128)
		for i := 0; i < 128; i++ {
			full[i] = rune(i)
		}
		neg := out[:0]
		for _, r := range full {
			if _, ok := set[r]; !ok {
				neg = append(neg, r)
			}
		}
		return neg, nil
	}
	return out, nil
}

func (p *parser) parseRepeat() (int, int, error) {
	p.scan() // '{'
	num := ""
	for p.look.typ == tChar && p.look.ch >= '0' && p.look.ch <= '9' {
		num += string(p.look.ch)
		p.scan()
	}
	if num == "" {
		return 0, 0, fmt.Errorf("expected number")
	}
	min, _ := strconv.Atoi(num)
	max := min
	if p.look.typ == tComma {
		p.scan()
		num = ""
		for p.look.typ == tChar && p.look.ch >= '0' && p.look.ch <= '9' {
			num += string(p.look.ch)
			p.scan()
		}
		if num == "" {
			max = -1
		} else {
			max, _ = strconv.Atoi(num)
		}
	}
	if p.look.typ != tRBrace {
		return 0, 0, fmt.Errorf("expected }")
	}
	p.scan()
	return min, max, nil
}
