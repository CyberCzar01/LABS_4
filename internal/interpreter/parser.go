package interpreter

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alecthomas/participle/v2"
	"github.com/alecthomas/participle/v2/lexer"
)

type Program struct {
	Statements []*Statement `parser:"@@*"`
}

type Statement struct {
	Decl   *Decl   `parser:"@@ ';'"`
	Assign *Assign `parser:"| @@ ';'"`
	Move   *Move   `parser:"| @@ ';'"`
	Loop   *Loop   `parser:"| @@"`
	If     *If     `parser:"| @@"`
}

type Decl struct {
	Type string `parser:"@( 'seisu' | 'ronri' )"`
	Name string `parser:"@Ident"`
	Expr *Expr  `parser:"( '=' @@ )?"`
}

type Assign struct {
	Name string `parser:"@Ident"`
	Expr *Expr  `parser:"'=' @@"`
}

type Move struct {
	Dir string `parser:"@Move"`
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
	Number *IntValue  `parser:"@Number"`
	Bool   *BoolValue `parser:"| @Bool"`
	Ident  *string    `parser:"| @Ident"`
}

type IntValue int

func (i *IntValue) Capture(values []string) error {
	s := values[0]
	base := 10
	if strings.HasPrefix(s, "x") || strings.HasPrefix(s, "X") {
		base = 16
		s = s[1:]
	}
	v, err := strconv.ParseInt(s, base, 64)
	if err != nil {
		return err
	}
	*i = IntValue(v)
	return nil
}

type BoolValue bool

func (b *BoolValue) Capture(values []string) error {
	switch values[0] {
	case "shinri":
		*b = true
	case "uso":
		*b = false
	default:
		return fmt.Errorf("invalid bool %s", values[0])
	}
	return nil
}

var langLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Move", Pattern: `\^_\^|v_v|<_<|>_>|o_o|~_~`},
	{Name: "Number", Pattern: `[-+]?(?:\d+|x[0-9A-F]+)`},
	{Name: "Bool", Pattern: `shinri|uso`},
	Number *int    `parser:"@Number"`
	Ident  *string `parser:"| @Ident"`
}

var langLexer = lexer.MustSimple([]lexer.SimpleRule{
	{Name: "Move", Pattern: `\^_\^|v_v|<_<|>_>|o_o|~_~`},
	{Name: "Number", Pattern: `\d+`},
	{Name: "Ident", Pattern: `[a-zA-Z_][a-zA-Z0-9_]*`},
	{Name: "Punct", Pattern: `[=;:+-]`},
	{Name: "whitespace", Pattern: `\s+`},
})

var parser = participle.MustBuild[Program](
	participle.Lexer(langLexer),
)

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
	case s.Decl != nil:
		val := 0
		if s.Decl.Expr != nil {
			var err error
			val, err = s.Decl.Expr.Eval(ctx)
			if err != nil {
				return err
			}
		}
		ctx.Env.Set(s.Decl.Name, val)
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
		if ctx.Lab != nil {
			ctx.Lab.Display(ctx.Robot)
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
		return int(*t.Number), nil
	case t.Bool != nil:
		if bool(*t.Bool) {
			return 1, nil
		}
		return 0, nil
	case t.Ident != nil:
		v, ok := ctx.Env.Get(*t.Ident)
		if !ok {
			return 0, fmt.Errorf("undefined variable %s", *t.Ident)
		}
		return v, nil
	}
	return 0, fmt.Errorf("invalid term")
}
