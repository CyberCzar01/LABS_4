package ast

import (
	"bytes"
	"fmt"
	"strings"

	"drone-maze/lexer"
)

type Node interface {
	TokenLiteral() string //первое слово первого оператора
	String() string       // проходит по дочерним узлам и склеивает их строки
}

type Statement interface {
	Node
	statementNode()
}

type Expression interface {
	Node
	expressionNode()
}

type Program struct {
	Statements []Statement
}

func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

func (p *Program) String() string {
	var out bytes.Buffer

	for _, s := range p.Statements {
		out.WriteString(s.String())
	}

	return out.String()
}

type ExpressionStatement struct {
	Token      lexer.Token
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

type VariableDeclaration struct {
	Token lexer.Token
	Type  string
	Name  *Identifier
	Value Expression
}

func (vd *VariableDeclaration) statementNode()       {}
func (vd *VariableDeclaration) TokenLiteral() string { return vd.Token.Literal }
func (vd *VariableDeclaration) String() string {
	var out bytes.Buffer

	out.WriteString(vd.TokenLiteral() + " ")
	out.WriteString(vd.Name.String())
	out.WriteString(" = ")

	if vd.Value != nil {
		out.WriteString(vd.Value.String())
	}

	out.WriteString(";")
	return out.String()
}

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) expressionNode()      {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) String() string {
	var out bytes.Buffer

	out.WriteString("{")
	for _, s := range bs.Statements {
		out.WriteString(s.String())
	}
	out.WriteString("}")

	return out.String()
}

type ReturnStatement struct {
	Token       lexer.Token
	ReturnValue Expression
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) String() string {
	var out bytes.Buffer

	out.WriteString(rs.TokenLiteral() + " ")

	if rs.ReturnValue != nil {
		out.WriteString(rs.ReturnValue.String())
	}

	out.WriteString(";")
	return out.String()
}

// оператор kido
type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) String() string {
	return bs.TokenLiteral() + ";"
}

// оператор shushi
type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) String() string {
	return cs.TokenLiteral() + ";"
}

// объявление функции
type FunctionDeclaration struct {
	Token      lexer.Token
	Name       *Identifier
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fd *FunctionDeclaration) statementNode()       {}
func (fd *FunctionDeclaration) expressionNode()      {}
func (fd *FunctionDeclaration) TokenLiteral() string { return fd.Token.Literal }
func (fd *FunctionDeclaration) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fd.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fd.TokenLiteral())
	out.WriteString(" ")
	out.WriteString(fd.Name.String())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fd.Body.String())

	return out.String()
}

// литерал функции
type FunctionLiteral struct {
	Token      lexer.Token // ТОкен kansu
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fl *FunctionLiteral) expressionNode()      {}
func (fl *FunctionLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FunctionLiteral) String() string {
	var out bytes.Buffer

	params := []string{}
	for _, p := range fl.Parameters {
		params = append(params, p.String())
	}

	out.WriteString(fl.TokenLiteral())
	out.WriteString("(")
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fl.Body.String())

	return out.String()
}

// идентификатор
type Identifier struct {
	Token lexer.Token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) String() string       { return i.Value }

// целочисленный литерал
type IntegerLiteral struct {
	Token lexer.Token
	Value int64
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// 16-ричный литерал
type HexLiteral struct {
	Token lexer.Token
	Value int64
}

func (hl *HexLiteral) expressionNode()      {}
func (hl *HexLiteral) TokenLiteral() string { return hl.Token.Literal }
func (hl *HexLiteral) String() string       { return hl.Token.Literal }

// логический литерал
type Boolean struct {
	Token lexer.Token
	Value bool
}

func (b *Boolean) expressionNode()      {}
func (b *Boolean) TokenLiteral() string { return b.Token.Literal }
func (b *Boolean) String() string       { return b.Token.Literal }

// префиксное выражение (-, ~)
type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(pe.Operator)
	out.WriteString(pe.Right.String())
	out.WriteString(")")

	return out.String()
}

// инфиксное выражение (+, *, <)
type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString(" " + ie.Operator + " ")
	out.WriteString(ie.Right.String())
	out.WriteString(")")

	return out.String()
}

// присваивание значения переменной
type AssignmentExpression struct {
	Token lexer.Token // Токен =
	Name  *Identifier
	Value Expression
}

func (ae *AssignmentExpression) expressionNode()      {}
func (ae *AssignmentExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AssignmentExpression) String() string {
	var out bytes.Buffer

	out.WriteString(ae.Name.String())
	out.WriteString(" = ")
	out.WriteString(ae.Value.String())

	return out.String()
}

// присваивание значения в индекс или поле
type SetExpression struct {
	Token  lexer.Token // Токен =
	Target Expression  // IndexExpression или FieldAccessExpression
	Value  Expression
}

func (se *SetExpression) expressionNode()      {}
func (se *SetExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SetExpression) String() string {
	var out bytes.Buffer
	out.WriteString(se.Target.String())
	out.WriteString(" = ")
	out.WriteString(se.Value.String())
	return out.String()
}

// условный оператор sorenara
type IfExpression struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (ie *IfExpression) expressionNode()      {}
func (ie *IfExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IfExpression) String() string {
	var out bytes.Buffer

	out.WriteString("sorenara ")
	out.WriteString(ie.Condition.String())
	out.WriteString(" ")
	out.WriteString(ie.Consequence.String())

	if ie.Alternative != nil {
		out.WriteString(" sorenara ")
		out.WriteString(ie.Alternative.String())
	}

	return out.String()
}

// цикл shuki
type LoopExpression struct {
	Token    lexer.Token
	Variable *Identifier
	Start    Expression
	End      Expression
	Body     *BlockStatement
}

func (le *LoopExpression) expressionNode()      {}
func (le *LoopExpression) TokenLiteral() string { return le.Token.Literal }
func (le *LoopExpression) String() string {
	var out bytes.Buffer

	out.WriteString("shuki ")
	out.WriteString(le.Variable.String())
	out.WriteString(" = ")
	out.WriteString(le.Start.String())
	out.WriteString(" : ")
	out.WriteString(le.End.String())
	out.WriteString(" ")
	out.WriteString(le.Body.String())

	return out.String()
}

// вызов функции
type CallExpression struct {
	Token     lexer.Token
	Function  Expression // Identifier или FunctionLiteral
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) String() string {
	var out bytes.Buffer
	args := []string{}
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	out.WriteString(ce.Function.String())
	out.WriteString("(")
	out.WriteString(strings.Join(args, ", "))
	out.WriteString(")")
	return out.String()
}

// литерал массива
type ArrayLiteral struct {
	Token    lexer.Token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) String() string {
	var out bytes.Buffer

	elements := []string{}
	for _, el := range al.Elements {
		elements = append(elements, el.String())
	}

	out.WriteString("[")
	out.WriteString(strings.Join(elements, ", "))
	out.WriteString("]")

	return out.String()
}

// обращение к элементу массива
type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) String() string {
	var out bytes.Buffer

	out.WriteString("(")
	out.WriteString(ie.Left.String())
	out.WriteString("[")
	out.WriteString(ie.Index.String())
	out.WriteString("])")

	return out.String()
}

// кубическая ячейка
type CellLiteral struct {
	Token      lexer.Token
	X          Expression
	Y          Expression
	Z          Expression
	IsObstacle Expression
}

func (cl *CellLiteral) expressionNode()      {}
func (cl *CellLiteral) TokenLiteral() string { return cl.Token.Literal }
func (cl *CellLiteral) String() string {
	var x, y, z, isObstacle string
	if cl.X != nil {
		x = cl.X.String()
	}
	if cl.Y != nil {
		y = cl.Y.String()
	}
	if cl.Z != nil {
		z = cl.Z.String()
	}
	if cl.IsObstacle != nil {
		isObstacle = cl.IsObstacle.String()
	}
	return fmt.Sprintf("{%s, %s, %s, %s}", x, y, z, isObstacle)
}

// доступ к полю (=>)
type FieldAccessExpression struct {
	Token     lexer.Token
	Object    Expression
	FieldName *Identifier
}

func (fae *FieldAccessExpression) expressionNode()      {}
func (fae *FieldAccessExpression) TokenLiteral() string { return fae.Token.Literal }
func (fae *FieldAccessExpression) String() string {
	return fmt.Sprintf("(%s=>%s)", fae.Object.String(), fae.FieldName.String())
}

// оператор ruikei
type RuikeiExpression struct {
	Token lexer.Token
	Left  Expression
	Right Expression
}

func (re *RuikeiExpression) expressionNode()      {}
func (re *RuikeiExpression) TokenLiteral() string { return re.Token.Literal }
func (re *RuikeiExpression) String() string {
	if re.Right == nil {
		return fmt.Sprintf("ruikei {%s}", re.Left.String())
	}
	return fmt.Sprintf("ruikei {%s %s}", re.Left.String(), re.Right.String())
}

type TypeNameLiteral struct {
	Token lexer.Token
	Value string
}

func (tnl *TypeNameLiteral) expressionNode()      {}
func (tnl *TypeNameLiteral) TokenLiteral() string { return tnl.Token.Literal }
func (tnl *TypeNameLiteral) String() string       { return tnl.Token.Literal }

// оператор jigen
type JigenExpression struct {
	Token lexer.Token
	Array Expression
}

func (je *JigenExpression) expressionNode()      {}
func (je *JigenExpression) TokenLiteral() string { return je.Token.Literal }
func (je *JigenExpression) String() string       { return "jigen " + je.Array.String() }

// >_>
type RobotOperation struct {
	Token lexer.Token
	Type  string
}

func (ro *RobotOperation) expressionNode()      {}
func (ro *RobotOperation) TokenLiteral() string { return ro.Token.Literal }
func (ro *RobotOperation) String() string       { return ro.Type }

// последовательность операторов
type SequenceExpression struct {
	Token      lexer.Token
	Operations []Expression
}

func (se *SequenceExpression) expressionNode()      {}
func (se *SequenceExpression) TokenLiteral() string { return se.Token.Literal }
func (se *SequenceExpression) String() string {
	var out bytes.Buffer

	out.WriteString("{")
	operations := []string{}
	for _, op := range se.Operations {
		operations = append(operations, op.String())
	}
	out.WriteString(strings.Join(operations, "; "))
	out.WriteString("}")

	return out.String()
}

// выражение со стрелкой
type ArrowExpression struct {
	Token lexer.Token
	Right Expression
}

func (ae *ArrowExpression) expressionNode()      {}
func (ae *ArrowExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *ArrowExpression) String() string {
	var out bytes.Buffer

	out.WriteString("=>")
	out.WriteString(ae.Right.String())

	return out.String()
}

// оператор объявления переменной
type LetStatement struct {
	Token lexer.Token // токен 'sentensu'
	Name  *Identifier
	Value Expression
}

func (ls *LetStatement) statementNode()       {}
func (ls *LetStatement) TokenLiteral() string { return ls.Token.Literal }
func (ls *LetStatement) String() string {
	var out bytes.Buffer

	out.WriteString(ls.TokenLiteral() + " ")
	out.WriteString(ls.Name.String())
	out.WriteString(" = ")

	if ls.Value != nil {
		out.WriteString(ls.Value.String())
	}

	out.WriteString(";")

	return out.String()
}

// объявление размерностей массива hairetsu
type ArrayDimensionsLiteral struct {
	Token lexer.Token // '{'
	Dims  []Expression
}

func (adl *ArrayDimensionsLiteral) expressionNode()      {}
func (adl *ArrayDimensionsLiteral) TokenLiteral() string { return adl.Token.Literal }
func (adl *ArrayDimensionsLiteral) String() string {
	parts := []string{}
	for _, d := range adl.Dims {
		parts = append(parts, d.String())
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

type MultiIndexExpression struct {
	Token   lexer.Token // '['
	Left    Expression
	Indexes []Expression
}

func (mie *MultiIndexExpression) expressionNode()      {}
func (mie *MultiIndexExpression) TokenLiteral() string { return mie.Token.Literal }
func (mie *MultiIndexExpression) String() string {
	parts := []string{}
	for _, idx := range mie.Indexes {
		parts = append(parts, idx.String())
	}
	return fmt.Sprintf("(%s[%s])", mie.Left.String(), strings.Join(parts, ", "))
}
