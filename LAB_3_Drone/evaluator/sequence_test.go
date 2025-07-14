package evaluator

import (
	"drone-maze/maze"
	"drone-maze/types"
	"testing"
)

func TestSequenceReturnsLast(t *testing.T) {
	input := `{1; 2; 3}`
	evaluated := testEval(t, input)
	testIntegerObject(t, evaluated, 3)
}

func TestSequenceBreak(t *testing.T) {
	InitState(maze.NewMaze())

	input := `{1; >_<; 5}`
	evaluated := testEval(t, input)
	testNullObject(t, evaluated)
}

func TestSequenceConditionalBreak(t *testing.T) {
	m := maze.NewMaze()
	m.SetCell(types.Cell{X: 0, Y: 1, Z: 0, IsObstacle: true})
	InitState(m)

	input := `{ ^_0; >_<; 42 }`
	evaluated := testEval(t, input)
	testNullObject(t, evaluated)
}
