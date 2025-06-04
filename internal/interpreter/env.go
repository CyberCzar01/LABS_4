package interpreter

import "fmt"

// Environment holds variables

type Environment struct {
	vars map[string]int
}

func NewEnvironment() *Environment {
	return &Environment{vars: make(map[string]int)}
}

func (e *Environment) Get(name string) (int, bool) {
	v, ok := e.vars[name]
	return v, ok
}

func (e *Environment) Set(name string, val int) {
	e.vars[name] = val
}

func (e *Environment) String() string {
	return fmt.Sprint(e.vars)
}
