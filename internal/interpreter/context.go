package interpreter

// Context stores environment and robot

type Context struct {
	Env   *Environment
	Robot *Robot
	Lab   *Labyrinth
}
