package interpreter

import (
	"fmt"

	"github.com/alecthomas/participle/v2"
)

type Program struct {
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Assign *Assign `parser:"@@ ';'"`
	Move   *Move   `parser:"| @@ ';'"`
	Loop   *Loop   `parser:"| @@"`
	If     *If     `parser:"| @@"`
}

type Assign struct {
	Name string `parser:"@Ident"`
	Expr *Expr  `parser:"'=' @@"`
}

type Move struct {
	Dir string `parser:"@('^_^'|'v_v'|'<_<'|'>_>'|'o_o'|'~_~')"`
}

type Loop struct {
	Var  string   `parser:"'shuki' @Ident"`
	From *Expr    `parser:"'=' @@ ':'"`
	To   *Expr    `parser:"@@"`
	Body *Program `parser:"'kido' @@ 'shushi'"`
}

type If struct {
	Cond *Expr    `parser:"'sorenara' @@"`
	Body *Program `parser:"'kido' @@ 'shushi'"`
}

type Expr struct {
	Left *Term     `parser:"@@"`
	Rest []*OpTerm `parser:"@@*"`
}

type OpTerm struct {
	Op    string `parser:"@('+'|'-')"`
	Right *Term  `parser:"@@"`
}

type Term struct {
	Number *int    `parser:"@Int"`
	Ident  *string `parser:"| @Ident"`
}

var parser = participle.MustBuild[Program]()

func Parse(data string) (*Program, error) {
	return parser.ParseString("input", data)
}

func (p *Program) Exec(ctx *Context) error {
	for _, stmt := range p.Statements {
		if err := stmt.Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

func (s *Statement) Exec(ctx *Context) error {
	switch {
	case s.Assign != nil:
		val, err := s.Assign.Expr.Eval(ctx)
		if err != nil {
			return err
		}
		ctx.Env.Set(s.Assign.Name, val)
	case s.Move != nil:
		switch s.Move.Dir {
		case "^_^":
			ctx.Robot.Move(0, 0, 1)
		case "v_v":
			ctx.Robot.Move(0, 0, -1)
		case "<_<":
			ctx.Robot.Move(-1, 0, 0)
		case ">_>":
			ctx.Robot.Move(1, 0, 0)
		case "o_o":
			ctx.Robot.Move(0, 1, 0)
		case "~_~":
			ctx.Robot.Move(0, -1, 0)
		}
	case s.Loop != nil:
		start, err := s.Loop.From.Eval(ctx)
		if err != nil {
			return err
		}
		end, err := s.Loop.To.Eval(ctx)
		if err != nil {
			return err
		}
		for i := start; i <= end; i++ {
			ctx.Env.Set(s.Loop.Var, i)
			if err := s.Loop.Body.Exec(ctx); err != nil {
				return err
			}
		}
	case s.If != nil:
		cond, err := s.If.Cond.Eval(ctx)
		if err != nil {
			return err
		}
		if cond != 0 {
			return s.If.Body.Exec(ctx)
		}
	}
	return nil
}

func (e *Expr) Eval(ctx *Context) (int, error) {
	val, err := e.Left.Eval(ctx)
	if err != nil {
		return 0, err
	}
	for _, rt := range e.Rest {
		v, err := rt.Right.Eval(ctx)
		if err != nil {
			return 0, err
		}
		switch rt.Op {
		case "+":
			val += v
		case "-":
			val -= v
		}
	}
	return val, nil
}

func (t *Term) Eval(ctx *Context) (int, error) {
	switch {
	case t.Number != nil:
		return *t.Number, nil
	case t.Ident != nil:
		v, ok := ctx.Env.Get(*t.Ident)
		if !ok {
			return 0, fmt.Errorf("undefined variable %s", *t.Ident)
		}
		return v, nil
	}
	return 0, fmt.Errorf("invalid term")
}
