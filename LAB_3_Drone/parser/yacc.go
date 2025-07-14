//go:generate goyacc -o y.go -p "yy" parser.y

package parser

import (
	"drone-maze/ast"
	"drone-maze/lexer"
	"drone-maze/token"
	"fmt"
)

var lex *LexerWrapper

type LexerWrapper struct {
	l      *lexer.Lexer
	result *ast.Program
	errors []Error
}

func NewLexerWrapper(l *lexer.Lexer) *LexerWrapper {
	return &LexerWrapper{
		l:      l,
		errors: []Error{},
	}
}

func (lx *LexerWrapper) Lex(yylval *yySymType) int {
	tok := lx.l.NextToken()
	yylval.token = tok

	if tok.Type == token.TOKEN_EOF {
		return 0 // 0 is EOF for yyParse
	}

	return lx.mapToken(tok.Type)
}

func (lx *LexerWrapper) mapToken(t token.Type) int {
	switch t {
	case token.TOKEN_EOF:
		return TOKEN_EOF
	case token.TOKEN_ILLEGAL:
		return TOKEN_ILLEGAL
	case token.TOKEN_SEISU:
		return TOKEN_SEISU
	case token.TOKEN_RONRI:
		return TOKEN_RONRI
	case token.TOKEN_RIPPOTAI:
		return TOKEN_RIPPOTAI
	case token.TOKEN_HAIRETSU:
		return TOKEN_HAIRETSU
	case token.TOKEN_KANSU:
		return TOKEN_KANSU
	case token.TOKEN_SHUKI:
		return TOKEN_SHUKI
	case token.TOKEN_SORENARA:
		return TOKEN_SORENARA
	case token.TOKEN_KIDO:
		return TOKEN_KIDO
	case token.TOKEN_SHUSHI:
		return TOKEN_SHUSHI
	case token.TOKEN_MODORU:
		return TOKEN_MODORU
	case token.TOKEN_RUIKEI:
		return TOKEN_RUIKEI
	case token.TOKEN_JIGEN:
		return TOKEN_JIGEN
	case token.TOKEN_SHINRI:
		return TOKEN_SHINRI
	case token.TOKEN_USO:
		return TOKEN_USO
	case token.TOKEN_IDENTIFIER:
		return TOKEN_IDENTIFIER
	case token.TOKEN_INTEGER:
		return TOKEN_INTEGER
	case token.TOKEN_HEX:
		return TOKEN_HEX
	case token.TOKEN_ASSIGN:
		return TOKEN_ASSIGN
	case token.TOKEN_PLUS:
		return TOKEN_PLUS
	case token.TOKEN_MINUS:
		return TOKEN_MINUS
	case token.TOKEN_ASTERISK:
		return TOKEN_ASTERISK
	case token.TOKEN_SLASH:
		return TOKEN_SLASH
	case token.TOKEN_NOT:
		return TOKEN_NOT
	case token.TOKEN_LESS:
		return TOKEN_LESS
	case token.TOKEN_GREATER:
		return TOKEN_GREATER
	case token.TOKEN_EQUALS:
		return TOKEN_EQUALS
	case token.TOKEN_NOTEQUALS:
		return TOKEN_NOTEQUALS
	case token.TOKEN_AND:
		return TOKEN_AND
	case token.TOKEN_OR:
		return TOKEN_OR
	case token.TOKEN_ARROW:
		return TOKEN_ARROW
	case token.TOKEN_PERCENT:
		return TOKEN_PERCENT
	case token.TOKEN_COMMA:
		return TOKEN_COMMA
	case token.TOKEN_SEMICOLON:
		return TOKEN_SEMICOLON
	case token.TOKEN_COLON:
		return TOKEN_COLON
	case token.TOKEN_LPAREN:
		return TOKEN_LPAREN
	case token.TOKEN_RPAREN:
		return TOKEN_RPAREN
	case token.TOKEN_LBRACE:
		return TOKEN_LBRACE
	case token.TOKEN_RBRACE:
		return TOKEN_RBRACE
	case token.TOKEN_LBRACKET:
		return TOKEN_LBRACKET
	case token.TOKEN_RBRACKET:
		return TOKEN_RBRACKET
	case token.TOKEN_MOVE_UP:
		return TOKEN_MOVE_UP
	case token.TOKEN_MOVE_DOWN:
		return TOKEN_MOVE_DOWN
	case token.TOKEN_MOVE_LEFT:
		return TOKEN_MOVE_LEFT
	case token.TOKEN_MOVE_RIGHT:
		return TOKEN_MOVE_RIGHT
	case token.TOKEN_MOVE_FORWARD:
		return TOKEN_MOVE_FORWARD
	case token.TOKEN_MOVE_BACKWARD:
		return TOKEN_MOVE_BACKWARD
	case token.TOKEN_MEASURE_UP:
		return TOKEN_MEASURE_UP
	case token.TOKEN_MEASURE_DOWN:
		return TOKEN_MEASURE_DOWN
	case token.TOKEN_MEASURE_LEFT:
		return TOKEN_MEASURE_LEFT
	case token.TOKEN_MEASURE_RIGHT:
		return TOKEN_MEASURE_RIGHT
	case token.TOKEN_MEASURE_FORWARD:
		return TOKEN_MEASURE_FORWARD
	case token.TOKEN_MEASURE_BACKWARD:
		return TOKEN_MEASURE_BACKWARD
	case token.TOKEN_GET_POSITION:
		return TOKEN_GET_POSITION
	case token.TOKEN_BREAK_SEQUENCE:
		return TOKEN_BREAK_SEQUENCE
	default:
		return TOKEN_ILLEGAL
	}
}

func (lx *LexerWrapper) Error(s string) {
	lx.errors = append(lx.errors, Error{Message: s, Line: -1, Column: -1}) // Position info not available yet
}

func Parse(l *lexer.Lexer) (*ast.Program, []Error) {
	lex = NewLexerWrapper(l)
	yyParse(lex)
	return lex.result, lex.errors
}

type Error struct {
	Message string
	Line    int
	Column  int
}

func (e Error) String() string {
	if e.Line > 0 {
		return fmt.Sprintf("parser error at line %d:%d: %s", e.Line, e.Column, e.Message)
	}
	return fmt.Sprintf("parser error: %s", e.Message)
}
