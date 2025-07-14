package evaluator

import (
	"drone-maze/lexer"
	"drone-maze/object"
	"drone-maze/parser"
	"strings"
	"testing"
)

func testEvalVerbose(t *testing.T, input string) object.Object {
	t.Helper()
	t.Logf("--- Testing Input ---\n%s\n---------------------", input)

	l, err := lexer.New([]byte(input))
	if err != nil {
		t.Fatalf("Lexer error: %v", err)
	}

	program, errors := parser.Parse(l)
	if len(errors) != 0 {
		var errorMessages []string
		for _, msg := range errors {
			errorMessages = append(errorMessages, msg.String())
		}
		t.Fatalf("Parser has %d errors:\n%s", len(errors), strings.Join(errorMessages, "\n"))
	}

	t.Logf("AST: %s", program.String())

	env := object.NewEnvironment()
	evaluated := Eval(program, env)

	if evaluated != nil {
		t.Logf("Result: [%s] %s", evaluated.Type(), evaluated.Inspect())
	} else {
		t.Logf("Result: <nil>")
	}

	return evaluated
}

func TestStrictAssignTyping(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{
		{"seisu a = shinri;", "type mismatch: cannot assign RONRI to SEISU"},
		{"ronri b = 10;", "type mismatch: cannot assign SEISU to RONRI"},
		{"rippotai c = 5;", "type mismatch: cannot assign SEISU to RIPPOTAI"},
		{"hairetsu d = shinri;", "type mismatch: cannot assign RONRI to HAIRETSU"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalVerbose(t, tt.input)
			errObj, ok := evaluated.(*object.Error)
			if !ok {
				t.Errorf("expected an error object, but got %T (%+v)", evaluated, evaluated)
				return
			}
			if errObj.Message != tt.expectedMessage {
				t.Errorf("wrong error message.\nexpected: %q\ngot:      %q", tt.expectedMessage, errObj.Message)
			}
		})
	}
}

func TestStrictArithmeticTyping(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{
		{"5 + shinri;", "type mismatch: SEISU + RONRI"},
		{"uso * 10;", "type mismatch: RONRI * SEISU"},
		{"[1] + [2];", "unknown operator: HAIRETSU + HAIRETSU"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			evaluated := testEvalVerbose(t, tt.input)
			errObj, ok := evaluated.(*object.Error)
			if !ok {
				t.Errorf("expected an error object, but got %T (%+v)", evaluated, evaluated)
				return
			}
			if errObj.Message != tt.expectedMessage {
				t.Errorf("wrong error message.\nexpected: %q\ngot:      %q", tt.expectedMessage, errObj.Message)
			}
		})
	}
}
