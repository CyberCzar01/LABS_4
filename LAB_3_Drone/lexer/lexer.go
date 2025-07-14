package lexer

import (
	"drone-maze/token"

	"github.com/timtadh/lexmachine"
	"github.com/timtadh/lexmachine/machines"
)

type Token struct {
	Type    token.Type
	Literal string
	Line    int
	Column  int
}

type Lexer struct {
	lexmachine *lexmachine.Lexer
	scanner    *lexmachine.Scanner
	input      []byte
	lastToken  token.Type
}

func New(input []byte) (*Lexer, error) {
	lexmachineLexer := lexmachine.NewLexer()
	lexmachineLexer.Add([]byte(`[ \t\n\r]+`), skip)
	lexmachineLexer.Add([]byte(`//[^\n]*`), skip)
	lexmachineLexer.Add([]byte(`[\\^]_[\\^]`), tokAction(token.TOKEN_MOVE_UP))
	lexmachineLexer.Add([]byte(`v_v`), tokAction(token.TOKEN_MOVE_DOWN))
	lexmachineLexer.Add([]byte(`<_<`), tokAction(token.TOKEN_MOVE_LEFT))
	lexmachineLexer.Add([]byte(`>_>`), tokAction(token.TOKEN_MOVE_RIGHT))
	lexmachineLexer.Add([]byte(`o_o`), tokAction(token.TOKEN_MOVE_FORWARD))
	lexmachineLexer.Add([]byte(`~_~`), tokAction(token.TOKEN_MOVE_BACKWARD))
	lexmachineLexer.Add([]byte(`[\\^]_0`), tokAction(token.TOKEN_MEASURE_UP))
	lexmachineLexer.Add([]byte(`v_0`), tokAction(token.TOKEN_MEASURE_DOWN))
	lexmachineLexer.Add([]byte(`<_0`), tokAction(token.TOKEN_MEASURE_LEFT))
	lexmachineLexer.Add([]byte(`>_0`), tokAction(token.TOKEN_MEASURE_RIGHT))
	lexmachineLexer.Add([]byte(`o_0`), tokAction(token.TOKEN_MEASURE_FORWARD))
	lexmachineLexer.Add([]byte(`~_0`), tokAction(token.TOKEN_MEASURE_BACKWARD))
	lexmachineLexer.Add([]byte(`[*]_[*]`), tokAction(token.TOKEN_GET_POSITION))
	lexmachineLexer.Add([]byte(`>_<`), tokAction(token.TOKEN_BREAK_SEQUENCE))
	lexmachineLexer.Add([]byte(`=>`), tokAction(token.TOKEN_ARROW))
	lexmachineLexer.Add([]byte(`==`), tokAction(token.TOKEN_EQUALS))
	lexmachineLexer.Add([]byte(`!=`), tokAction(token.TOKEN_NOTEQUALS))
	lexmachineLexer.Add([]byte(`=`), tokAction(token.TOKEN_ASSIGN))
	lexmachineLexer.Add([]byte(`[+]`), tokAction(token.TOKEN_PLUS))
	lexmachineLexer.Add([]byte(`-`), tokAction(token.TOKEN_MINUS))
	lexmachineLexer.Add([]byte(`[*]`), tokAction(token.TOKEN_ASTERISK))
	lexmachineLexer.Add([]byte(`/`), tokAction(token.TOKEN_SLASH))
	lexmachineLexer.Add([]byte(`%`), tokAction(token.TOKEN_PERCENT))
	lexmachineLexer.Add([]byte(`~`), tokAction(token.TOKEN_NOT))
	lexmachineLexer.Add([]byte(`<`), tokAction(token.TOKEN_LESS))
	lexmachineLexer.Add([]byte(`>`), tokAction(token.TOKEN_GREATER))
	lexmachineLexer.Add([]byte(`[\\^]`), tokAction(token.TOKEN_AND))
	lexmachineLexer.Add([]byte(`v`), tokAction(token.TOKEN_OR))
	lexmachineLexer.Add([]byte(`,`), tokAction(token.TOKEN_COMMA))
	lexmachineLexer.Add([]byte(`;`), tokAction(token.TOKEN_SEMICOLON))
	lexmachineLexer.Add([]byte(`:`), tokAction(token.TOKEN_COLON))
	lexmachineLexer.Add([]byte(`[(]`), tokAction(token.TOKEN_LPAREN))
	lexmachineLexer.Add([]byte(`[)]`), tokAction(token.TOKEN_RPAREN))
	lexmachineLexer.Add([]byte(`[{]`), tokAction(token.TOKEN_LBRACE))
	lexmachineLexer.Add([]byte(`[}]`), tokAction(token.TOKEN_RBRACE))
	lexmachineLexer.Add([]byte(`[\[]`), tokAction(token.TOKEN_LBRACKET))
	lexmachineLexer.Add([]byte(`[]]`), tokAction(token.TOKEN_RBRACKET))
	keywords := map[string]token.Type{
		"seisu":    token.TOKEN_SEISU,
		"ronri":    token.TOKEN_RONRI,
		"rippotai": token.TOKEN_RIPPOTAI,
		"hairetsu": token.TOKEN_HAIRETSU,
		"kansu":    token.TOKEN_KANSU,
		"shuki":    token.TOKEN_SHUKI,
		"sorenara": token.TOKEN_SORENARA,
		"kido":     token.TOKEN_KIDO,
		"shushi":   token.TOKEN_SHUSHI,
		"modoru":   token.TOKEN_MODORU,
		"ruikei":   token.TOKEN_RUIKEI,
		"jigen":    token.TOKEN_JIGEN,
		"shinri":   token.TOKEN_SHINRI,
		"uso":      token.TOKEN_USO,
	}
	for keyword, tokenType := range keywords {
		lexmachineLexer.Add([]byte(keyword), tokAction(tokenType))
	}

	lexmachineLexer.Add([]byte(`0[xX][0-9A-Fa-f]+`), tokAction(token.TOKEN_INTEGER))
	lexmachineLexer.Add([]byte(`0|[1-9][0-9]*`), tokAction(token.TOKEN_INTEGER))
	lexmachineLexer.Add([]byte(`[a-zA-Z_][a-zA-Z0-9_]*`), tokAction(token.TOKEN_IDENTIFIER))

	err := lexmachineLexer.Compile()
	if err != nil {
		return nil, err
	}

	scanner, err := lexmachineLexer.Scanner(input)
	if err != nil {
		return nil, err
	}

	return &Lexer{
		lexmachine: lexmachineLexer,
		scanner:    scanner,
		input:      input,
		lastToken:  token.TOKEN_ILLEGAL,
	}, nil
}

func (l *Lexer) NextToken() Token {
	tok, err, eof := l.scanner.Next()

	if eof {
		return Token{Type: token.TOKEN_EOF}
	}
	if err != nil {
		return Token{Type: token.TOKEN_ILLEGAL, Literal: err.Error()}
	}

	tokInterface := tok.(Token)
	l.lastToken = tokInterface.Type
	return tokInterface
}

func skip(*lexmachine.Scanner, *machines.Match) (interface{}, error) {
	return nil, nil
}

func tokAction(tokenType token.Type) lexmachine.Action {
	return func(s *lexmachine.Scanner, m *machines.Match) (interface{}, error) {
		return Token{
			Type:    tokenType,
			Literal: string(m.Bytes),
			Line:    m.StartLine,
			Column:  m.StartColumn,
		}, nil
	}
}
