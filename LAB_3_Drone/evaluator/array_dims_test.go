package evaluator

import (
	"drone-maze/object"
	"testing"
)

func TestMultiDimensionalArray(t *testing.T) {
	input := `
hairetsu a = {2, 3};
a[1, 2] = 7;
a[1,2];
`
	evaluated := testEval(t, input)
	testIntegerObject(t, evaluated, 7)
}

func TestJigen(t *testing.T) {
	input := `hairetsu a = {2,3,4}; jigen a;`
	evaluated := testEval(t, input)
	testIntegerObject(t, evaluated, 3)
}

func TestArrayIndexOutOfBounds(t *testing.T) {
	input := `hairetsu a = {2,3}; a[2,0];`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error object for out of bounds, got %T", evaluated)
	}
}
