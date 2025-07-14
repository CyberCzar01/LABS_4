%{
package parser

import (
	"drone-maze/ast"
	"drone-maze/lexer"
	"fmt"
	"strconv"
)

%}

%union {
	token			lexer.Token
	program			*ast.Program
	statement		ast.Statement
	expression		ast.Expression
	blockStatement	*ast.BlockStatement
	identifier		*ast.Identifier
	statements		[]ast.Statement
	expressions		[]ast.Expression
	identifiers		[]*ast.Identifier
	str_val			string
	int_val			int64
	ast            ast.Node
}

%token <token> TOKEN_EOF
%token <token> TOKEN_ILLEGAL

%token <token> TOKEN_SEISU TOKEN_RONRI TOKEN_RIPPOTAI TOKEN_HAIRETSU TOKEN_KANSU
%token <token> TOKEN_SHUKI TOKEN_SORENARA TOKEN_KIDO TOKEN_SHUSHI TOKEN_MODORU
%token <token> TOKEN_RUIKEI TOKEN_JIGEN TOKEN_SHINRI TOKEN_USO

%token <token> TOKEN_IDENTIFIER TOKEN_INTEGER TOKEN_HEX

%token <token> TOKEN_ASSIGN TOKEN_PLUS TOKEN_MINUS TOKEN_ASTERISK TOKEN_SLASH
%token <token> TOKEN_NOT TOKEN_LESS TOKEN_GREATER TOKEN_EQUALS TOKEN_NOTEQUALS
%token <token> TOKEN_AND TOKEN_OR TOKEN_ARROW TOKEN_PERCENT

%token <token> TOKEN_COMMA TOKEN_SEMICOLON TOKEN_COLON
%token <token> TOKEN_LPAREN TOKEN_RPAREN TOKEN_LBRACE TOKEN_RBRACE
%token <token> TOKEN_LBRACKET TOKEN_RBRACKET

%token <token> TOKEN_MOVE_UP TOKEN_MOVE_DOWN TOKEN_MOVE_LEFT TOKEN_MOVE_RIGHT
%token <token> TOKEN_MOVE_FORWARD TOKEN_MOVE_BACKWARD
%token <token> TOKEN_MEASURE_UP TOKEN_MEASURE_DOWN TOKEN_MEASURE_LEFT TOKEN_MEASURE_RIGHT
%token <token> TOKEN_MEASURE_FORWARD TOKEN_MEASURE_BACKWARD
%token <token> TOKEN_GET_POSITION TOKEN_BREAK_SEQUENCE

%type <program> program
%type <statements> statement_list
%type <statement> statement
%type <expression> expression assignment_expression
%type <expression> simple_expression literal robot_operation
%type <expression> if_expression loop_expression array_literal cell_literal function_literal ruikei_expression jigen_expression cell_literal_body
%type <statement> variable_declaration return_statement break_statement continue_statement function_declaration expression_statement
%type <blockStatement> block_statement if_tail
%type <identifier> identifier
%type <expressions> expression_list non_empty_expression_list
%type <identifiers> parameter_list non_empty_parameter_list
%type <token> variable_type
%type <expression> array_dims_literal
%type <expression> sequence_expression operation_item
%type <expressions> operation_list optional_operation_list

%right TOKEN_ASSIGN
%left  TOKEN_OR TOKEN_AND
%left  TOKEN_EQUALS TOKEN_NOTEQUALS
%left  TOKEN_LESS TOKEN_GREATER
%left  TOKEN_PLUS TOKEN_MINUS
%left  TOKEN_ASTERISK TOKEN_SLASH TOKEN_PERCENT
%right TOKEN_NOT
%left  TOKEN_LPAREN TOKEN_LBRACKET TOKEN_ARROW

%%

program:
	statement_list {
		lex.result = &ast.Program{Statements: $1}
	}
;

statement_list:
	/* empty */ {
		$$ = []ast.Statement{}
	}
|	statement_list statement {
		$$ = append($1, $2)
	}
;

statement:
	variable_declaration { $$ = $1 }
|	return_statement { $$ = $1 }
|	break_statement { $$ = $1 }
|	continue_statement { $$ = $1 }
|	function_declaration { $$ = $1 }
|	expression_statement { $$ = $1 }
;

expression_statement:
	expression optional_semicolon {
		$$ = &ast.ExpressionStatement{Token: lexer.Token{}, Expression: $1}
	}
;

optional_semicolon:
	/* empty */
|	TOKEN_SEMICOLON
;

variable_declaration:
	variable_type TOKEN_IDENTIFIER TOKEN_ASSIGN expression optional_semicolon {
		ident := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.VariableDeclaration{Token: $1, Type: $1.Literal, Name: ident, Value: $4}
	}
|	TOKEN_RIPPOTAI TOKEN_IDENTIFIER TOKEN_ASSIGN expression optional_semicolon {
		ident := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.VariableDeclaration{Token: $1, Type: $1.Literal, Name: ident, Value: $4}
	}
|	TOKEN_RIPPOTAI TOKEN_IDENTIFIER TOKEN_ASSIGN cell_literal_body optional_semicolon {
		ident := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.VariableDeclaration{Token: $1, Type: $1.Literal, Name: ident, Value: $4}
	}
|	TOKEN_HAIRETSU TOKEN_IDENTIFIER TOKEN_ASSIGN expression optional_semicolon {
		ident := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.VariableDeclaration{Token: $1, Type: $1.Literal, Name: ident, Value: $4}
	}
;

variable_type:
	TOKEN_SEISU { $$ = $1 }
|	TOKEN_RONRI { $$ = $1 }
|	TOKEN_HAIRETSU { $$ = $1 }
;

return_statement:
	TOKEN_MODORU expression optional_semicolon {
		$$ = &ast.ReturnStatement{Token: $1, ReturnValue: $2}
	}
;

break_statement:
	TOKEN_KIDO optional_semicolon {
		$$ = &ast.BreakStatement{Token: $1}
	}
;

continue_statement:
	TOKEN_SHUSHI optional_semicolon {
		$$ = &ast.ContinueStatement{Token: $1}
	}
;

function_declaration:
	TOKEN_KANSU TOKEN_IDENTIFIER TOKEN_LPAREN parameter_list TOKEN_RPAREN block_statement {
		name := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.FunctionDeclaration{Token: $1, Name: name, Parameters: $4, Body: $6}
	}
;

block_statement:
	TOKEN_LBRACE statement_list TOKEN_RBRACE {
		$$ = &ast.BlockStatement{Token: $1, Statements: $2}
	}
|	TOKEN_KIDO statement_list TOKEN_SHUSHI {
		$$ = &ast.BlockStatement{Token: $1, Statements: $2}
	}
;


expression:
	simple_expression { $$ = $1 }
|	assignment_expression { $$ = $1 }
|	expression TOKEN_PLUS expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "+", Right: $3} }
|	expression TOKEN_MINUS expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "-", Right: $3} }
|	expression TOKEN_ASTERISK expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "*", Right: $3} }
|	expression TOKEN_SLASH expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "/", Right: $3} }
|	expression TOKEN_PERCENT expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "%", Right: $3} }
|	expression TOKEN_EQUALS expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "==", Right: $3} }
|	expression TOKEN_NOTEQUALS expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "!=", Right: $3} }
|	expression TOKEN_LESS expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "<", Right: $3} }
|	expression TOKEN_GREATER expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: ">", Right: $3} }
|	expression TOKEN_AND expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "^", Right: $3} }
|	expression TOKEN_OR expression { $$ = &ast.InfixExpression{Token: $2, Left: $1, Operator: "v", Right: $3} }
|	TOKEN_MINUS expression %prec TOKEN_NOT { $$ = &ast.PrefixExpression{Token: $1, Operator: "-", Right: $2} }
|	TOKEN_NOT expression { $$ = &ast.PrefixExpression{Token: $1, Operator: "~", Right: $2} }
|	if_expression { $$ = $1 }
|	loop_expression { $$ = $1 }
|	TOKEN_LPAREN expression TOKEN_RPAREN { $$ = $2 }
|	expression TOKEN_LPAREN expression_list TOKEN_RPAREN { $$ = &ast.CallExpression{Token: $2, Function: $1, Arguments: $3} }
|	expression TOKEN_LBRACKET non_empty_expression_list TOKEN_RBRACKET {
		if len($3)==1 {
			$$ = &ast.IndexExpression{Token: $2, Left: $1, Index: $3[0]}
		} else {
			$$ = &ast.MultiIndexExpression{Token: $2, Left: $1, Indexes: $3}
		}
	}
|	expression TOKEN_LBRACKET non_empty_expression_list TOKEN_RBRACKET TOKEN_ASSIGN expression {
		if len($3)==1 {
			target := &ast.IndexExpression{Token: $2, Left: $1, Index: $3[0]}
			$$ = &ast.SetExpression{Token: $5, Target: target, Value: $6}
		} else {
			target := &ast.MultiIndexExpression{Token: $2, Left: $1, Indexes: $3}
			$$ = &ast.SetExpression{Token: $5, Target: target, Value: $6}
		}
	}
|	expression TOKEN_ARROW identifier { $$ = &ast.FieldAccessExpression{Token: $2, Object: $1, FieldName: $3} }
|	sequence_expression { $$ = $1 }
;

assignment_expression:
	identifier TOKEN_ASSIGN expression {
		$$ = &ast.AssignmentExpression{Token: $2, Name: $1, Value: $3}
	}
;


simple_expression:
	literal { $$ = $1 }
|	identifier { $$ = $1 }
|	array_literal { $$ = $1 }
|	cell_literal { $$ = $1 }
|	function_literal { $$ = $1 }
|	ruikei_expression { $$ = $1 }
|	jigen_expression { $$ = $1 }
|	robot_operation { $$ = $1 }
|	array_dims_literal { $$ = $1 }
;

literal:
	TOKEN_INTEGER {
		val, err := strconv.ParseInt($1.Literal, 10, 64)
		if err != nil {
			lex.Error(fmt.Sprintf("could not parse %q as integer: %v", $1.Literal, err))
		}
		$$ = &ast.IntegerLiteral{Token: $1, Value: val}
	}
|	TOKEN_HEX {
		val, err := strconv.ParseInt($1.Literal[1:], 16, 64) // skip 'x'
		if err != nil {
			lex.Error(fmt.Sprintf("could not parse %q as hex: %v", $1.Literal, err))
		}
		$$ = &ast.HexLiteral{Token: $1, Value: val}
	}
|	TOKEN_SHINRI { $$ = &ast.Boolean{Token: $1, Value: true} }
|	TOKEN_USO    { $$ = &ast.Boolean{Token: $1, Value: false} }
;

identifier:
	TOKEN_IDENTIFIER { $$ = &ast.Identifier{Token: $1, Value: $1.Literal} }
;

array_literal:
	TOKEN_LBRACKET expression_list TOKEN_RBRACKET {
		$$ = &ast.ArrayLiteral{Token: $1, Elements: $2}
	}
;

cell_literal:
	TOKEN_RIPPOTAI cell_literal_body {
		// a cell literal is just the body, the token is consumed to decide the rule
		$$ = $2
	}
;

cell_literal_body:
	TOKEN_LBRACE expression TOKEN_COMMA expression TOKEN_COMMA expression TOKEN_COMMA expression TOKEN_RBRACE {
		$$ = &ast.CellLiteral{Token: $1, X: $2, Y: $4, Z: $6, IsObstacle: $8}
	}
;

function_literal:
	TOKEN_KANSU TOKEN_LPAREN parameter_list TOKEN_RPAREN block_statement {
		$$ = &ast.FunctionLiteral{Token: $1, Parameters: $3, Body: $5}
	}
;

parameter_list:
	/* empty */ { $$ = []*ast.Identifier{} }
|	non_empty_parameter_list { $$ = $1 }
;

non_empty_parameter_list:
	identifier { $$ = []*ast.Identifier{$1} }
|	non_empty_parameter_list TOKEN_COMMA identifier { $$ = append($1, $3) }
;

expression_list:
	/* empty */ { $$ = []ast.Expression{} }
|	non_empty_expression_list { $$ = $1 }
;

non_empty_expression_list:
	expression { $$ = []ast.Expression{$1} }
|	non_empty_expression_list TOKEN_COMMA expression { $$ = append($1, $3) }
;

if_expression:
	TOKEN_SORENARA TOKEN_LPAREN expression TOKEN_RPAREN block_statement if_tail {
		$$ = &ast.IfExpression{Token: $1, Condition: $3, Consequence: $5, Alternative: $6}
	}
;

if_tail:
	/* empty */ { $$ = nil }
|	TOKEN_SORENARA block_statement { $$ = $2 }
|	TOKEN_SORENARA TOKEN_LPAREN expression TOKEN_RPAREN block_statement if_tail {
		inner := &ast.IfExpression{Token: $1, Condition: $3, Consequence: $5, Alternative: $6}
		$$ = &ast.BlockStatement{Token: $1, Statements: []ast.Statement{&ast.ExpressionStatement{Expression: inner}}}
	}
;

loop_expression:
	TOKEN_SHUKI TOKEN_IDENTIFIER TOKEN_ASSIGN expression TOKEN_COLON expression block_statement {
		ident := &ast.Identifier{Token: $2, Value: $2.Literal}
		$$ = &ast.LoopExpression{Token: $1, Variable: ident, Start: $4, End: $6, Body: $7}
	}
;

ruikei_expression:
	TOKEN_RUIKEI TOKEN_LBRACE expression TOKEN_RBRACE {
		$$ = &ast.RuikeiExpression{Token: $1, Left: $3, Right: nil}
	}
|	TOKEN_RUIKEI TOKEN_LBRACE expression expression TOKEN_RBRACE {
		$$ = &ast.RuikeiExpression{Token: $1, Left: $3, Right: $4}
	}
;

jigen_expression:
	TOKEN_JIGEN expression {
		$$ = &ast.JigenExpression{Token: $1, Array: $2}
	}
;

robot_operation:
	TOKEN_MOVE_UP { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MOVE_DOWN { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MOVE_LEFT { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MOVE_RIGHT { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MOVE_FORWARD { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MOVE_BACKWARD { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_UP { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_DOWN { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_LEFT { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_RIGHT { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_FORWARD { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_MEASURE_BACKWARD { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_GET_POSITION { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
|	TOKEN_BREAK_SEQUENCE { $$ = &ast.RobotOperation{Token: $1, Type: $1.Literal} }
;

sequence_expression:
	TOKEN_LBRACE optional_operation_list TOKEN_RBRACE {
		$$ = &ast.SequenceExpression{Token: $1, Operations: $2}
	}
;

optional_operation_list:
	/* empty */ { $$ = []ast.Expression{} }
|	operation_list { $$ = $1 }
;

operation_list:
	operation_item { $$ = []ast.Expression{$1} }
|	operation_list TOKEN_SEMICOLON operation_item { $$ = append($1, $3) }
|	operation_list TOKEN_SEMICOLON { $$ = $1 } // allow trailing semicolon
;

operation_item:
	expression { $$ = $1 }
;

array_dims_literal:
	TOKEN_LBRACE non_empty_expression_list TOKEN_RBRACE {
		$$ = &ast.ArrayDimensionsLiteral{Token: $1, Dims: $2}
	}
;

%%