package evaluator

import (
	"drone-maze/maze"
	"drone-maze/object"
	"drone-maze/types"
	"testing"
)

func TestArrayNegativeDimension(t *testing.T) {
	input := `hairetsu bad = {-1};`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error for negative dimension, got %T", evaluated)
	}
}

func TestArrayWrongIndexCount(t *testing.T) {
	input := `hairetsu a = {2,3}; a[1];`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error for wrong index count, got %T", evaluated)
	}
}

func TestArrayMultiIndexOutOfBounds(t *testing.T) {
	input := `hairetsu a = {2,3}; a[1,3];`
	evaluated := testEval(t, input)
	if _, ok := evaluated.(*object.Error); !ok {
		t.Errorf("expected error for out-of-bounds, got %T", evaluated)
	}
}

func TestNestedSequenceConditionalBreak(t *testing.T) {
	m := maze.NewMaze()
	m.SetCell(types.Cell{X: 0, Y: 0, Z: 1, IsObstacle: true})
	InitState(m)

	input := `{ { o_0; >_< }; 99 }`
	evaluated := testEval(t, input)
	testNullObject(t, evaluated)
}
