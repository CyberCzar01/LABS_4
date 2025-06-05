"""
quadro.lexer  –  лексический анализатор Quadro (Stage 3-a / блок C-D)
Совместимо с Python 3.9 + PLY 3.11.
"""

import ply.lex as lex

# --------------------- пропускаем комментарии --------------------------
t_ignore_COMMENT = r'\#.*'

# --------------------- пробелы и табуляция ----------------------------
t_ignore = ' \t\r'

# --------------------- токены -------------------------------------------------
tokens = (
    # ключевые слова
    'SEISU', 'RONRI', 'RIPPOTAI', 'HAIRETSU',
    'SORENARA', 'KIDO', 'SHUSHI', 'SHUKI',
    'KANSU', 'JIGEN', 'RUIKEI', 'WHERE',

    # литералы
    'INT_DEC', 'INT_HEX', 'SHINRI_LIT', 'USO_LIT',

    # операторы и символы
    'PLUS', 'MINUS', 'TIMES', 'TILDE', 'CARET', 'V_OP',
    'LT', 'GT', 'ASSIGN',
    'COMMA', 'SEMICOLON', 'COLON',
    'LPAREN', 'RPAREN', 'LBRACE', 'RBRACE',
    'ARROW',

    # команды робота
    'MOVE_UP', 'MOVE_DOWN', 'MOVE_LEFT', 'MOVE_RIGHT', 'MOVE_FWD', 'MOVE_BACK',
    'MEAS_UP', 'MEAS_DOWN', 'MEAS_LEFT', 'MEAS_RIGHT', 'MEAS_FWD', 'MEAS_BACK',
    'STOP_IF',

    # идентификаторы
    'IDENT',
)

# --------------------- регулярки для символов ----------------------------
t_PLUS      = r'\+'
t_MINUS     = r'-'
t_TIMES     = r'\*'
t_TILDE     = r'~'
t_CARET     = r'\^'
t_V_OP      = r'v'
t_LT        = r'<'
t_GT        = r'>'
t_ASSIGN    = r'='
t_COMMA     = r','
t_SEMICOLON = r';'
t_COLON     = r':'
t_LPAREN    = r'\('
t_RPAREN    = r'\)'
t_LBRACE    = r'\{'
t_RBRACE    = r'\}'
t_ARROW     = r'=>'

# --------------------- команды робота в токенах ----------------------------
t_MOVE_UP    = r'\^_\^'
t_MOVE_DOWN  = r'v_v'
t_MOVE_LEFT  = r'<_<'
t_MOVE_RIGHT = r'>_>'
t_MOVE_FWD   = r'o_o'
t_MOVE_BACK  = r'~_~'

t_MEAS_UP    = r'\^_0'
t_MEAS_DOWN  = r'v_0'
t_MEAS_LEFT  = r'<_0'
t_MEAS_RIGHT = r'>_0'
t_MEAS_FWD   = r'o_0'
t_MEAS_BACK  = r'~_0'

t_STOP_IF    = r'>_<'
t_WHERE      = r'\*_\*'

# --------------------- ключевые слова и идентификаторы --------------------
reserved = {
    'seisu':    'SEISU',
    'ronri':    'RONRI',
    'rippotai': 'RIPPOTAI',
    'hairetsu': 'HAIRETSU',
    'sorenara': 'SORENARA',
    'kido':     'KIDO',
    'shushi':   'SHUSHI',
    'shuki':    'SHUKI',
    'kansu':    'KANSU',
    'jigen':    'JIGEN',
    'ruikei':   'RUIKEI',
    'shinri':   'SHINRI_LIT',
    'uso':      'USO_LIT',
}

def t_INT_HEX(t):
    r'x[0-9A-F]+'
    # удаляем префикс 'x', парсим как hex
    t.value = int(t.value[1:], 16)
    return t

def t_INT_DEC(t):
    r'\d+'
    t.value = int(t.value, 10)
    return t

def t_IDENT(t):
    r'[A-Za-z_][A-Za-z0-9_]*'
    t.type = reserved.get(t.value, 'IDENT')
    return t

# --------------------- новая строка (для подсчёта lineno) -----------------
def t_newline(t):
    r'\n+'
    t.lexer.lineno += len(t.value)

# --------------------- ошибка лексера ------------------------------------
def t_error(t):
    raise SyntaxError(f"Illegal char {t.value[0]!r} line {t.lineno}")

# ---------------------------------------------------------------------------
lexer = lex.lex()
