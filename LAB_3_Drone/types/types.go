package types

// ячейка
type Cell struct {
	X, Y, Z    int
	IsObstacle bool
}

// робот
type Robot struct {
	Position  Cell
	IsBroken  bool
	Direction Direction
}

// напрвление движения
type Direction int

const (
	Up Direction = iota
	Down
	Left
	Right
	Forward
	Backward
)

// значение
type Value interface {
	Type() string
}

// целое значение
type IntegerValue struct {
	Value int64
}

func (v IntegerValue) Type() string {
	return "seisu"
}

// true-false
type BooleanValue struct {
	Value bool
}

func (v BooleanValue) Type() string {
	return "ronri"
}

// значение ячейки
type CellValue struct {
	Value Cell
}

func (v CellValue) Type() string {
	return "rippotai"
}

// массив
type ArrayValue struct {
	Values []Value
	Dims   []int
}

func (v ArrayValue) Type() string {
	return "hairetsu"
}
