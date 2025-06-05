import ply.yacc as yacc
from lexer import tokens, lexer
from ast import *
from semantic import semantic_check

# --------------------- Приоритеты ----------------------------
precedence = (
    ('left', 'V_OP'),
    ('left', 'CARET_OP'),
    ('right', 'TILDE_OP'),
    ('left', 'LT_CMP', 'GT_CMP'),
    ('left', 'PLUS', 'MINUS'),
    ('left', 'TIMES'),
)

# --------------------- Грамматика ----------------------------

def p_program(p):
    'program : top_list'
    p[0] = ProgramNode(p[1])

def p_top_list_single(p):
    'top_list : top'
    p[0] = [p[1]]

def p_top_list_append(p):
    'top_list : top_list top'
    p[0] = p[1] + [p[2]]

def p_top_decl_list(p):
    'top : decl_list'
    p[0] = TopDeclsNode(p[1])

def p_top_func(p):
    'top : function_def'
    p[0] = TopFuncNode(p[1])

def p_top_block(p):
    'top : block_stmt'
    p[0] = TopBlockNode(p[1])

# --------------------- Объявления ----------------------------

def p_decl_list_single(p):
    'decl_list : decl SEMICOLON'
    p[0] = [p[1]]

def p_decl_list_append(p):
    'decl_list : decl_list decl SEMICOLON'
    p[0] = p[1] + [p[2]]

def p_decl_int(p):
    'decl : SEISU_KW IDENT opt_assign_expr'
    p[0] = VarDeclNode('seisu', p[2], p[3])

def p_decl_bool(p):
    'decl : RONRI_KW IDENT opt_assign_bool'
    p[0] = VarDeclNode('ronri', p[2], p[3])

def p_decl_cell(p):
    'decl : RIPPOTAI_KW IDENT ASSIGN LBRACE expr COMMA expr COMMA expr COMMA bool_expr RBRACE'
    p[0] = CellDeclNode(p[2], p[5], p[7], p[9], p[11])

def p_decl_array(p):
    'decl : HAIRETSU_KW IDENT ASSIGN LBRACE array_dims RBRACE'
    p[0] = ArrayDeclNode(p[2], p[5])

def p_opt_assign_expr_empty(p):
    'opt_assign_expr : '
    p[0] = None

def p_opt_assign_expr(p):
    'opt_assign_expr : ASSIGN expr'
    p[0] = p[2]

def p_opt_assign_bool_empty(p):
    'opt_assign_bool : '
    p[0] = None

def p_opt_assign_bool(p):
    'opt_assign_bool : ASSIGN bool_expr'
    p[0] = p[2]

def p_array_dims_one(p):
    'array_dims : expr'
    p[0] = [p[1]]

def p_array_dims_app(p):
    'array_dims : array_dims COMMA expr'
    p[0] = p[1] + [p[3]]

# --------------------- Доступ к полю ячейки ----------------------
def p_field_access(p):
    'field_access : IDENT ASSIGN GT_CMP IDENT'
    p[0] = CellFieldNode(p[1], p[4])

# --------------------- L-value ------------------------------
def p_lvalue_ident(p):
    'lvalue : IDENT'
    p[0] = VarRefNode(p[1])

def p_lvalue_index(p):
    'lvalue : IDENT LBRACKET index_list RBRACKET'
    p[0] = ArrayAccessNode(p[1], p[3])

def p_index_list_one(p):
    'index_list : expr'
    p[0] = [p[1]]

def p_index_list_app(p):
    'index_list : index_list COMMA expr'
    p[0] = p[1] + [p[3]]

# --------------------- jigen и ruikei ------------------------
def p_jigen(p):
    'jigen_stmt : JIGEN_KW IDENT SEMICOLON'
    p[0] = JigenNode(p[2])

def p_ruikei(p):
    'ruikei_stmt : RUIKEI_KW LBRACE ruikei_item ruikei_item RBRACE SEMICOLON'
    p[0] = RuikeiNode(p[3], p[4])

def p_ruikei_item_ident(p):
    'ruikei_item : IDENT'
    p[0] = RuikeiItemIdentNode(p[1])

def p_ruikei_item_array(p):
    'ruikei_item : IDENT LBRACKET index_list RBRACKET'
    p[0] = RuikeiItemArrayNode(p[1], p[3])

def p_ruikei_item_type(p):
    '''ruikei_item : SEISU_KW
                   | RONRI_KW
                   | RIPPOTAI_KW
                   | HAIRETSU_KW'''
    p[0] = RuikeiItemTypeNode(p[1])

# --------------------- Выражения ------------------------
def p_expr_int(p):
    'expr : SEISU'
    p[0] = IntLiteralNode(p[1])

def p_expr_ident(p):
    'expr : IDENT'
    p[0] = VarRefNode(p[1])

def p_expr_array_access(p):
    'expr : lvalue'
    p[0] = ArrayAccessExprNode(p[1])

def p_expr_paren(p):
    'expr : LPAREN expr RPAREN'
    p[0] = p[2]

def p_expr_binop(p):
    '''expr : expr PLUS expr
            | expr MINUS expr
            | expr TIMES expr'''
    p[0] = ArithOpNode(p[2], p[1], p[3])

def p_bool_lit(p):
    'bool_expr : RONRI'
    p[0] = BoolLiteralNode(p[1])

def p_bool_paren(p):
    'bool_expr : LPAREN bool_expr RPAREN'
    p[0] = p[2]

def p_bool_unary(p):
    'bool_expr : TILDE_OP bool_expr'
    p[0] = BoolOpNode('NOT', None, p[2])

def p_bool_binop(p):
    '''bool_expr : bool_expr CARET_OP bool_expr
                 | bool_expr V_OP bool_expr'''
    op = 'AND' if p[2] == '^' else 'OR'
    p[0] = BoolOpNode(op, p[1], p[3])

def p_bool_cmp(p):
    '''bool_expr : expr LT_CMP expr
                 | expr GT_CMP expr'''
    p[0] = CmpOpNode(p[2], p[1], p[3])

# --------------------- Операторы ------------------------
def p_stmt(p):
    'stmt : simple_stmt SEMICOLON'
    p[0] = p[1]

def p_simple_assign(p):
    'simple_stmt : lvalue ASSIGN expr'
    p[0] = AssignNode(p[1], p[3])

def p_simple_field_access(p):
    'simple_stmt : field_access'
    p[0] = p[1]

def p_simple_jigen(p):
    'simple_stmt : jigen_stmt'
    p[0] = p[1]

def p_simple_ruikei(p):
    'simple_stmt : ruikei_stmt'
    p[0] = p[1]

def p_simple_move(p):
    '''simple_stmt : MOVE_UP
                   | MOVE_DOWN
                   | MOVE_LEFT
                   | MOVE_RIGHT
                   | MOVE_FWD
                   | MOVE_BACK'''
    dir_map = {
        '^_^': 'UP', 'v_v': 'DOWN',
        '<_<': 'LEFT', '>_>': 'RIGHT',
        'o_o': 'FWD', '~_~': 'BACK'
    }
    p[0] = MoveNode(dir_map[p[1]])

def p_simple_meas(p):
    '''simple_stmt : MEAS_UP
                   | MEAS_DOWN
                   | MEAS_LEFT
                   | MEAS_RIGHT
                   | MEAS_FWD
                   | MEAS_BACK'''
    dir_map = {
        '^_0': 'UP', 'v_0': 'DOWN',
        '<_0': 'LEFT', '>_0': 'RIGHT',
        'o_0': 'FWD', '~_0': 'BACK'
    }
    p[0] = MeasureNode(dir_map[p[1]])

def p_simple_stop_if(p):
    'simple_stmt : STOP_IF'
    p[0] = StopIfNode()

def p_simple_get_pos(p):
    'simple_stmt : GET_POS'
    p[0] = GetPosNode()

def p_simple_if(p):
    'simple_stmt : SORENARA bool_expr KIDO stmt_list SHUSHI'
    p[0] = IfNode(p[2], p[4])

def p_simple_for(p):
    'simple_stmt : SHUKI IDENT ASSIGN expr COLON expr KIDO stmt_list SHUSHI'
    p[0] = ForNode(p[2], p[4], p[6], p[8])

def p_simple_block(p):
    'simple_stmt : block_stmt'
    p[0] = p[1]

def p_simple_func_call(p):
    'simple_stmt : func_call'
    p[0] = p[1]

def p_block_stmt(p):
    'block_stmt : LBRACE stmt_list RBRACE'
    p[0] = BlockNode(p[2)]

def p_stmt_list_empty(p):
    'stmt_list : '
    p[0] = []

def p_stmt_list_append(p):
    'stmt_list : stmt_list stmt'
    p[0] = p[1] + [p[2]]

# --------------------- Описание функций ------------------------
def p_function_def(p):
    'function_def : type_spec KANSU IDENT LPAREN param_list RPAREN KIDO stmt_list SHUSHI'
    p[0] = FuncDefNode(p[1], p[3], p[5], p[8])

def p_type_spec(p):
    '''type_spec : SEISU_KW
                 | RONRI_KW
                 | RIPPOTAI_KW
                 | HAIRETSU_KW'''
    p[0] = p[1]

def p_param_list_empty(p):
    'param_list : '
    p[0] = []

def p_param_list_nonempty(p):
    'param_list : param_list_nonempty'
    p[0] = p[1]

def p_param_list_one(p):
    'param_list_nonempty : type_spec IDENT'
    p[0] = [(p[1], p[2])]

def p_param_list_app(p):
    'param_list_nonempty : param_list_nonempty COMMA type_spec IDENT'
    p[0] = p[1] + [(p[3], p[4])]

# --------------------- Вызов функции ------------------------
def p_func_call(p):
    'func_call : IDENT LPAREN arg_list RPAREN'
    p[0] = FuncCallNode(p[1], p[3])

def p_arg_list_empty(p):
    'arg_list : '
    p[0] = []

def p_arg_list_nonempty(p):
    'arg_list : arg_list_nonempty'
    p[0] = p[1]

def p_arg_list_one(p):
    'arg_list_nonempty : expr'
    p[0] = [p[1]]

def p_arg_list_app(p):
    'arg_list_nonempty : arg_list_nonempty COMMA expr'
    p[0] = p[1] + [p[3]]

# --------------------- Ошибка синтаксиса ------------------------
def p_error(p):
    if p:
        raise SyntaxError(f"syntax error: token={p.type} value={p.value!r} line={p.lineno}")
    else:
        raise SyntaxError("syntax error at EOF")

# --------------------- Генерация парсера ------------------------
parser = yacc.yacc(debug=False, write_tables=False)

def parse(src, **kwargs):
    ast = parser.parse(src, lexer=lexer, **kwargs)
    semantic_check(ast)
    return ast
