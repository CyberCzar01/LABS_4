package object

import (
	"bytes"
	"drone-maze/ast"
	"fmt"
	"strings"
)

type ObjectType string

const (
	INTEGER_OBJ      ObjectType = "SEISU"
	BOOLEAN_OBJ      ObjectType = "RONRI"
	NULL_OBJ         ObjectType = "NULL"
	RETURN_VALUE_OBJ ObjectType = "RETURN_VALUE"
	ERROR_OBJ        ObjectType = "ERROR"
	FUNCTION_OBJ     ObjectType = "KANSU"
	STRING_OBJ       ObjectType = "STRING"
	ARRAY_OBJ        ObjectType = "HAIRETSU"
	CELL_OBJ         ObjectType = "RIPPOTAI"
	BREAK_OBJ        ObjectType = "BREAK"
	CONTINUE_OBJ     ObjectType = "CONTINUE"
	TYPE_NAME_OBJ    ObjectType = "TYPE_NAME"
	ROBOT_OBJ        ObjectType = "ROBOT"
)

type Object interface {
	Type() ObjectType
	Inspect() string
}

type Integer struct {
	Value int64
}

func (i *Integer) Type() ObjectType { return INTEGER_OBJ }
func (i *Integer) Inspect() string  { return fmt.Sprintf("%d", i.Value) }

type Boolean struct {
	Value bool
}

func (b *Boolean) Type() ObjectType { return BOOLEAN_OBJ }
func (b *Boolean) Inspect() string  { return fmt.Sprintf("%t", b.Value) }

type Null struct{}

func (n *Null) Type() ObjectType { return NULL_OBJ }
func (n *Null) Inspect() string  { return "null" }

type ReturnValue struct {
	Value Object
}

func (rv *ReturnValue) Type() ObjectType { return RETURN_VALUE_OBJ }
func (rv *ReturnValue) Inspect() string  { return rv.Value.Inspect() }

type BreakValue struct{}

func (bv *BreakValue) Type() ObjectType { return BREAK_OBJ }
func (bv *BreakValue) Inspect() string  { return "break" }

type ContinueValue struct{}

func (cv *ContinueValue) Type() ObjectType { return CONTINUE_OBJ }
func (cv *ContinueValue) Inspect() string  { return "continue" }

type Error struct {
	Message string
}

func (e *Error) Type() ObjectType { return ERROR_OBJ }
func (e *Error) Inspect() string  { return "ERROR: " + e.Message }

type Environment struct {
	store map[string]Object
	outer *Environment
}

func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s, outer: nil}
}

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}

func (e *Environment) Update(name string, val Object) bool {
	_, ok := e.store[name]
	if ok {
		e.store[name] = val
		return true
	}
	if e.outer != nil {
		return e.outer.Update(name, val)
	}
	return false
}

func (e *Environment) Delete(name string) {
	delete(e.store, name)
}

type Function struct {
	Parameters []*ast.Identifier
	Body       *ast.BlockStatement
	Env        *Environment
}

func (f *Function) Type() ObjectType { return FUNCTION_OBJ }
func (f *Function) Inspect() string {
	var out bytes.Buffer
	params := []string{}
	for _, p := range f.Parameters {
		params = append(params, p.String())
	}
	out.WriteString("kansu")
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") {\n")
	out.WriteString(f.Body.String())
	out.WriteString("\n}")
	return out.String()
}

type String struct {
	Value string
}

func (s *String) Type() ObjectType { return STRING_OBJ }
func (s *String) Inspect() string  { return s.Value }

type Array struct {
	Elements   []Object
	Dimensions []int64
}

func (ao *Array) Type() ObjectType { return ARRAY_OBJ }
func (ao *Array) Inspect() string {
	var out bytes.Buffer
	dims := []string{}
	for _, d := range ao.Dimensions {
		dims = append(dims, fmt.Sprintf("%d", d))
	}
	out.WriteString(fmt.Sprintf("hairetsu<%s>", strings.Join(dims, ", ")))
	return out.String()
}

type Cell struct {
	X, Y, Z    Object
	IsObstacle Object
	IsExit     Object
}

func (c *Cell) Type() ObjectType { return CELL_OBJ }
func (c *Cell) Inspect() string {
	// Если IsExit не задан (nil), выводим только 4 поля для совместимости
	if c.IsExit == nil {
		return fmt.Sprintf("{%s, %s, %s, %s}", c.X.Inspect(), c.Y.Inspect(), c.Z.Inspect(), c.IsObstacle.Inspect())
	}
	return fmt.Sprintf("{%s, %s, %s, %s, %s}", c.X.Inspect(), c.Y.Inspect(), c.Z.Inspect(), c.IsObstacle.Inspect(), c.IsExit.Inspect())
}

type Robot struct {
	X, Y, Z  int
	IsBroken bool
}

func (r *Robot) Type() ObjectType { return ROBOT_OBJ }
func (r *Robot) Inspect() string {
	return fmt.Sprintf("Robot at (%d, %d, %d)", r.X, r.Y, r.Z)
}

type TypeName struct {
	Name string
}

func (tn *TypeName) Type() ObjectType { return TYPE_NAME_OBJ }
func (tn *TypeName) Inspect() string  { return tn.Name }
