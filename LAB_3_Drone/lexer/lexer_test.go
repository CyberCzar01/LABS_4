package lexer

import (
	"drone-maze/token"
	"testing"
)

func TestNextToken(t *testing.T) {
	input := `seisu five = 5;
seisu ten = 10;

seisu add = kansu(x, y) {
  x + y;
};

seisu result = add(five, ten);
~-/*5;
5 < 10 > 5;

sorenara (5 < 10) {
	modoru shinri;
} sorenara {
	modoru uso;
}

10 == 10;
10 != 9;
[1, 2];
^_^ v_v <_< >_> o_o ~_~
^_0 v_0 <_0 >_0 o_0 ~_0
*_* >_<
`
	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.TOKEN_SEISU, "seisu"},
		{token.TOKEN_IDENTIFIER, "five"},
		{token.TOKEN_ASSIGN, "="},
		{token.TOKEN_INTEGER, "5"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_SEISU, "seisu"},
		{token.TOKEN_IDENTIFIER, "ten"},
		{token.TOKEN_ASSIGN, "="},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_SEISU, "seisu"},
		{token.TOKEN_IDENTIFIER, "add"},
		{token.TOKEN_ASSIGN, "="},
		{token.TOKEN_KANSU, "kansu"},
		{token.TOKEN_LPAREN, "("},
		{token.TOKEN_IDENTIFIER, "x"},
		{token.TOKEN_COMMA, ","},
		{token.TOKEN_IDENTIFIER, "y"},
		{token.TOKEN_RPAREN, ")"},
		{token.TOKEN_LBRACE, "{"},
		{token.TOKEN_IDENTIFIER, "x"},
		{token.TOKEN_PLUS, "+"},
		{token.TOKEN_IDENTIFIER, "y"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_RBRACE, "}"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_SEISU, "seisu"},
		{token.TOKEN_IDENTIFIER, "result"},
		{token.TOKEN_ASSIGN, "="},
		{token.TOKEN_IDENTIFIER, "add"},
		{token.TOKEN_LPAREN, "("},
		{token.TOKEN_IDENTIFIER, "five"},
		{token.TOKEN_COMMA, ","},
		{token.TOKEN_IDENTIFIER, "ten"},
		{token.TOKEN_RPAREN, ")"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_NOT, "~"},
		{token.TOKEN_MINUS, "-"},
		{token.TOKEN_SLASH, "/"},
		{token.TOKEN_ASTERISK, "*"},
		{token.TOKEN_INTEGER, "5"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_INTEGER, "5"},
		{token.TOKEN_LESS, "<"},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_GREATER, ">"},
		{token.TOKEN_INTEGER, "5"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_SORENARA, "sorenara"},
		{token.TOKEN_LPAREN, "("},
		{token.TOKEN_INTEGER, "5"},
		{token.TOKEN_LESS, "<"},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_RPAREN, ")"},
		{token.TOKEN_LBRACE, "{"},
		{token.TOKEN_MODORU, "modoru"},
		{token.TOKEN_SHINRI, "shinri"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_RBRACE, "}"},
		{token.TOKEN_SORENARA, "sorenara"},
		{token.TOKEN_LBRACE, "{"},
		{token.TOKEN_MODORU, "modoru"},
		{token.TOKEN_USO, "uso"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_RBRACE, "}"},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_EQUALS, "=="},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_INTEGER, "10"},
		{token.TOKEN_NOTEQUALS, "!="},
		{token.TOKEN_INTEGER, "9"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_LBRACKET, "["},
		{token.TOKEN_INTEGER, "1"},
		{token.TOKEN_COMMA, ","},
		{token.TOKEN_INTEGER, "2"},
		{token.TOKEN_RBRACKET, "]"},
		{token.TOKEN_SEMICOLON, ";"},
		{token.TOKEN_MOVE_UP, "^_^"},
		{token.TOKEN_MOVE_DOWN, "v_v"},
		{token.TOKEN_MOVE_LEFT, "<_<"},
		{token.TOKEN_MOVE_RIGHT, ">_>"},
		{token.TOKEN_MOVE_FORWARD, "o_o"},
		{token.TOKEN_MOVE_BACKWARD, "~_~"},
		{token.TOKEN_MEASURE_UP, "^_0"},
		{token.TOKEN_MEASURE_DOWN, "v_0"},
		{token.TOKEN_MEASURE_LEFT, "<_0"},
		{token.TOKEN_MEASURE_RIGHT, ">_0"},
		{token.TOKEN_MEASURE_FORWARD, "o_0"},
		{token.TOKEN_MEASURE_BACKWARD, "~_0"},
		{token.TOKEN_GET_POSITION, "*_*"},
		{token.TOKEN_BREAK_SEQUENCE, ">_<"},
		{token.TOKEN_EOF, ""},
	}

	l, err := New([]byte(input))
	if err != nil {
		t.Fatalf("Failed to create lexer: %v", err)
	}

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. input=`%s`\nexpected type=%q (%d),\ngot type=%q (%d)",
				i, tt.expectedLiteral, tt.expectedType, tt.expectedType, tok.Type, tok.Type)
		}

		if tt.expectedType != token.TOKEN_ILLEGAL && tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. input=`%s`\nexpected literal=%q,\ngot literal=%q",
				i, tt.expectedLiteral, tt.expectedLiteral, tok.Literal)
		}
	}
}

// Helper to add a test for a single illegal character
func TestNextToken_Illegal(t *testing.T) {
	input := "@"
	l, _ := New([]byte(input))
	tok := l.NextToken()
	if tok.Type != token.TOKEN_ILLEGAL {
		t.Fatalf("tokentype wrong. expected=%q, got=%q", token.TOKEN_ILLEGAL, tok.Type)
	}
}
