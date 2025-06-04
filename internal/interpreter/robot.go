package interpreter

import "fmt"

// Robot represents robot position in 3D grid

type Robot struct {
	X, Y, Z int
}

func NewRobot() *Robot {
	return &Robot{}
}

func (r *Robot) Move(dx, dy, dz int) {
	r.X += dx
	r.Y += dy
	r.Z += dz
}

func (r *Robot) Position() (int, int, int) {
	return r.X, r.Y, r.Z
}

func (r *Robot) String() string {
	return fmt.Sprintf("(%d,%d,%d)", r.X, r.Y, r.Z)
}
