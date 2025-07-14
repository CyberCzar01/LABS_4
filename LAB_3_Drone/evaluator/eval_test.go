package evaluator

import (
	"drone-maze/lexer"
	"drone-maze/maze"
	"drone-maze/object"
	"drone-maze/parser"
	"drone-maze/types"
	"strings"
	"testing"
)

func TestEvalIntegerExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"5", 5},
		{"10", 10},
		{"-5", -5},
		{"-10", -10},
		{"5 + 5 + 5 + 5 - 10", 10},
		{"2 * 2 * 2 * 2 * 2", 32},
		{"-50 + 100 + -50", 0},
		{"5 * 2 + 10", 20},
		{"5 + 2 * 10", 25},
		{"20 + 2 * -10", 0},
		{"50 / 2 * 2 + 10", 60},
		{"2 * (5 + 10)", 30},
		{"3 * 3 * 3 + 10", 37},
		{"3 * (3 * 3) + 10", 37},
		{"(5 + 10 * 2 + 15 / 3) * 2 + -10", 50},
		{"0x10", 16},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestEvalBooleanExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"shinri", true},
		{"uso", false},
		{"1 < 2", true},
		{"1 > 2", false},
		{"1 < 1", false},
		{"1 > 1", false},
		{"1 == 1", true},
		{"1 != 1", false},
		{"1 == 2", false},
		{"1 != 2", true},
		{"shinri == shinri", true},
		{"uso == uso", true},
		{"shinri == uso", false},
		{"shinri != uso", true},
		{"(1 < 2) == shinri", true},
		{"(1 < 2) == uso", false},
		{"(1 > 2) == shinri", false},
		{"(1 > 2) == uso", true},
		{"shinri v uso", true},
		{"shinri ^ uso", false},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestNotOperator(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"~shinri", false},
		{"~uso", true},
		{"~5", false},
		{"~~shinri", true},
		{"~~uso", false},
		{"~~5", true},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testBooleanObject(t, evaluated, tt.expected)
	}
}

func TestIfElseExpressions(t *testing.T) {
	tests := []struct {
		input    string
		expected interface{}
	}{
		{"sorenara (shinri) { 10 }", 10},
		{"sorenara (uso) { 10 }", nil},
		{"sorenara (1) { 10 }", 10},
		{"sorenara (1 < 2) { 10 }", 10},
		{"sorenara (1 > 2) { 10 }", nil},
		{"sorenara (1 > 2) { 10 } sorenara { 20 }", 20},
		{"sorenara (1 < 2) { 10 } sorenara { 20 }", 10},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		integer, ok := tt.expected.(int)
		if ok {
			testIntegerObject(t, evaluated, int64(integer))
		} else {
			testNullObject(t, evaluated)
		}
	}
}

func TestReturnStatements(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"modoru 10;", 10},
		{"modoru 10; 9;", 10},
		{"modoru 2 * 5; 9;", 10},
		{"9; modoru 2 * 5; 9;", 10},
		{
			`
sorenara (10 > 1) {
  sorenara (10 > 1) {
    modoru 10;
  }
  modoru 1;
}
`,
			10,
		},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		testIntegerObject(t, evaluated, tt.expected)
	}
}

func TestErrorHandling(t *testing.T) {
	tests := []struct {
		input           string
		expectedMessage string
	}{

		{"-shinri", "unknown operator: -RONRI"},
		{"foobar", "identifier not found: foobar"},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		errObj, ok := evaluated.(*object.Error)
		if !ok {
			t.Errorf("no error object returned. got=%T(%+v)", evaluated, evaluated)
			continue
		}
		if errObj.Message != tt.expectedMessage {
			t.Errorf("wrong error message. expected=%q, got=%q", tt.expectedMessage, errObj.Message)
		}
	}
}

func TestVariableDeclaration(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"seisu a = 5; a;", 5},
		{"seisu a = 5 * 5; a;", 25},
		{"seisu a = 5; seisu b = a; b;", 5},
		{"seisu a = 5; seisu b = a; seisu c = a + b + 5; c;", 15},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestFunctionApplication(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"kansu identity(x) { x; } identity(5);", 5},
		{"kansu identity(x) { modoru x; } identity(5);", 5},
		{"kansu double(x) { x * 2; } double(5);", 10},
		{"kansu add(x, y) { x + y; } add(5, 5);", 10},
		{"kansu add(x, y) { x + y; } add(5 + 5, add(5, 5));", 20},
		{"kansu(x) { x; }(5)", 5},
	}

	for _, tt := range tests {
		testIntegerObject(t, testEval(t, tt.input), tt.expected)
	}
}

func TestClosures(t *testing.T) {
	input := `
kansu newAdder(x) {
  modoru kansu(y) { modoru x + y; };
}
newAdder(2)(3);
`
	testIntegerObject(t, testEval(t, input), 5)
}

func TestArrayLiterals(t *testing.T) {
	input := "[1, 2 * 2, 3 + 3]"
	evaluated := testEval(t, input)
	result, ok := evaluated.(*object.Array)
	if !ok {
		t.Fatalf("object is not Array. got=%T (%+v)", evaluated, evaluated)
	}
	if len(result.Elements) != 3 {
		t.Fatalf("array has wrong num of elements. got=%d", len(result.Elements))
	}
	testIntegerObject(t, result.Elements[0], 1)
	testIntegerObject(t, result.Elements[1], 4)
	testIntegerObject(t, result.Elements[2], 6)
}

func TestLoopExpressions(t *testing.T) {
	input := `
seisu x = 0;
seisu y = 0;
shuki i = 0:5 {
	y = y + 1;
	sorenara (i == 1) {
		shushi;
	}
	x = x + 1;
	sorenara (i == 3) {
		kido;
	}
}
// Returns a cell with the final values of x and y
rippotai{x, y, 0, uso}
`
	evaluated := testEval(t, input)
	cell, ok := evaluated.(*object.Cell)
	if !ok {
		t.Fatalf("expected Cell, got %T", evaluated)
	}
	testIntegerObject(t, cell.X, 3)
	testIntegerObject(t, cell.Y, 4)
}

func TestCellAndFieldAccess(t *testing.T) {
	input := `
rippotai c = {1, 2, 3, shinri};
c=>y
`
	evaluated := testEval(t, input)
	testIntegerObject(t, evaluated, 2)
}

func TestRobotOperations(t *testing.T) {
	testMaze := maze.NewMaze()
	testMaze.Robot = types.Robot{
		Position: types.Cell{X: 5, Y: 5, Z: 5},
		IsBroken: false,
	}
	InitState(testMaze)

	tests := []struct {
		input    string
		testFunc func(t *testing.T, obj object.Object)
	}{
		{">_>;", func(t *testing.T, obj object.Object) {
			robot := GetMaze().GetRobot()
			if robot.Position.X != 6 {
				t.Errorf("robot X pos should be 6, got %d", robot.Position.X)
			}
		}},
		{"*_*=>x", func(t *testing.T, obj object.Object) {
			testIntegerObject(t, obj, 6)
		}},
	}

	for _, tt := range tests {
		evaluated := testEval(t, tt.input)
		tt.testFunc(t, evaluated)
	}
}

func testEval(t *testing.T, input string) object.Object {
	t.Helper()
	l, err := lexer.New([]byte(input))
	if err != nil {
		t.Fatalf("Lexer error on input %q: %v", input, err)
	}
	program, errors := parser.Parse(l)
	if len(errors) != 0 {
		var errorMessages []string
		for _, msg := range errors {
			errorMessages = append(errorMessages, msg.String())
		}
		t.Fatalf("Parser has %d errors on input %q:\n%s", len(errors), input, strings.Join(errorMessages, "\n"))
	}
	env := object.NewEnvironment()
	evaluated := Eval(program, env)
	t.Logf("Input: %q, Evaluated: %+v", input, evaluated)
	return evaluated
}

func testIntegerObject(t *testing.T, obj object.Object, expected int64) bool {
	t.Helper()
	result, ok := obj.(*object.Integer)
	if !ok {
		t.Errorf("object is not Integer. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%d, want=%d", result.Value, expected)
		return false
	}
	return true
}

func testBooleanObject(t *testing.T, obj object.Object, expected bool) bool {
	t.Helper()
	result, ok := obj.(*object.Boolean)
	if !ok {
		t.Errorf("object is not Boolean. got=%T (%+v)", obj, obj)
		return false
	}
	if result.Value != expected {
		t.Errorf("object has wrong value. got=%t, want=%t", result.Value, expected)
		return false
	}
	return true
}

func testNullObject(t *testing.T, obj object.Object) bool {
	t.Helper()
	if obj != NULL {
		t.Errorf("object is not NULL. got=%T (%+v)", obj, obj)
		return false
	}
	return true
}
