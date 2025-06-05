"""
Строгий семантический анализ Quadro.
Проверяет:
• объявление до использования
• повторное объявление
• изменение константы
• согласованность типов в присваиваниях, выражениях, вызовах функций
"""

from __future__ import annotations

from typing import Dict, List, Optional
import importlib.util
import os
import sys

if __package__:
    from .runtime import IntVal, BoolVal, ArrayVal, CellVal, FunctionVal
else:
    from runtime import IntVal, BoolVal, ArrayVal, CellVal, FunctionVal



# ------------------ типы на уровне языка --------------------------------
INT, BOOL, CELL, ARRAY, FUNC = 'int', 'bool', 'cell', 'array', 'func'


def expr_type(node: ast.Node) -> str:
    """Возвращает строковый тип узла-выражения (исп. рантайм-классы)."""
    if isinstance(node, ast.IntLit):        return INT
    if isinstance(node, ast.BoolLit):       return BOOL
    if isinstance(node, ast.CellLit):       return CELL
    if isinstance(node, ast.ArrayLit):      return ARRAY
    if isinstance(node, ast.Ident):         return 'ident'   # определится позже
    if isinstance(node, ast.BinArith):      return INT
    if isinstance(node, ast.Compare):       return BOOL
    if isinstance(node, ast.Logical):       return BOOL
    if isinstance(node, ast.Not):           return BOOL
    if isinstance(node, ast.Ruikei):        return BOOL
    if isinstance(node, ast.CellField):     return INT if node.field in 'xyz' else BOOL
    if isinstance(node, ast.Jigen):         return ARRAY      # список размеров
    if isinstance(node, ast.FuncCall):      return INT  # вернём int; детализация ниже
    if isinstance(node, ast.WhereExpr):     return CELL
    if isinstance(node, ast.Sequence):      return ARRAY
    return 'unknown'


# ------------------ окружение типов ------------------------------------
class SymInfo:
    def __init__(self, type_: str, is_const: bool = False):
        self.type = type_
        self.is_const = is_const


class Scope:
    def __init__(self, parent: Optional['Scope'] = None):
        self.parent = parent
        self.table: Dict[str, SymInfo] = {}

    def declare(self, name: str, info: SymInfo):
        if name in self.table:
            raise SemanticError(f"Повторное объявление {name!r}")
        self.table[name] = info

    def lookup(self, name: str) -> SymInfo:
        if name in self.table:
            return self.table[name]
        if self.parent:
            return self.parent.lookup(name)
        raise SemanticError(f"Идентификатор {name!r} не объявлен")


# ------------------ исключение -----------------------------------------
class SemanticError(Exception):
    pass


# import local ast module after defining SemanticError to avoid circular import
if __package__:
    from . import ast
else:
    module_path = os.path.join(os.path.dirname(__file__), 'ast.py')
    spec = importlib.util.spec_from_file_location('quadro_ast', module_path)
    ast = importlib.util.module_from_spec(spec)
    sys.modules['quadro_ast'] = ast
    sys.modules['ast'] = ast
    spec.loader.exec_module(ast)

# ------------------ главный проход -------------------------------------
def check(program: ast.Program):
    walk(program, Scope())


def walk(node: ast.Node, scope: Scope) -> str:
    """
    Обходит дерево; возвращает тип узла (строкой).
    Бросает SemanticError при нарушении правил.
    """
    # ----- узлы верхнего уровня ---------------------------------------
    if isinstance(node, ast.Program) or isinstance(node, ast.Block):
        child_scope = Scope(scope) if isinstance(node, ast.Block) else scope
        for st in node.body if isinstance(node, ast.Program) else node.stmts:
            walk(st, child_scope)
        return 'void'

    # ----- объявления --------------------------------------------------
    if isinstance(node, ast.VarDecl):
        declared_type = deduce_decl_type(node)
        scope.declare(node.name, SymInfo(declared_type, node.is_const))
        if node.expr:
            rhs_type = walk(node.expr, scope)
            require_assignable(declared_type, rhs_type,
                               f"Несовместимо: {declared_type} := {rhs_type}")
        return 'void'

    # ----- присваивание -----------------------------------------------
    if isinstance(node, ast.Assign):
        info = scope.lookup(node.target)
        if info.is_const:
            raise SemanticError(f"Попытка изменить константу {node.target!r}")
        rhs_type = walk(node.expr, scope)
        require_assignable(info.type, rhs_type,
                           f"Несовместимо: {info.type} := {rhs_type}")
        return 'void'

    # ----- if / for ----------------------------------------------------
    if isinstance(node, ast.If):
        cond_t = walk(node.cond, scope)
        if cond_t != BOOL:
            raise SemanticError("Условие sorenara не bool")
        walk(node.body, Scope(scope))
        return 'void'

    if isinstance(node, ast.For):
        walk(node.start, scope); walk(node.end, scope)
        itype = INT   # переменная цикла всегда int
        scope.declare(node.var, SymInfo(itype))
        walk(node.body, Scope(scope))
        return 'void'

    # ----- функции -----------------------------------------------------
    if isinstance(node, ast.FuncDef):
        scope.declare(node.name, SymInfo(FUNC, is_const=True))
        fn_scope = Scope(scope)
        for p in node.param_names:
            fn_scope.declare(p, SymInfo(INT))      # упрощённо: все int
        walk(node.body, fn_scope)
        return 'void'

    if isinstance(node, ast.FuncCall):
        scope.lookup(node.name)    # убедимся, что объявлена
        for a in node.args:
            walk(a, scope)
        return INT

    # ----- робо-команды / Sequence ------------------------------------
    if isinstance(node, ast.MoveCmd) or isinstance(node, ast.MeasureCmd):
        return INT
    if isinstance(node, ast.Sequence):
        for c in node.cmds:
            walk(c, scope)
        return ARRAY
    if isinstance(node, ast.StopIf):
        return 'void'
    if isinstance(node, ast.WhereExpr):
        return CELL

    # ----- выражения ---------------------------------------------------
    if isinstance(node, ast.IntLit) or isinstance(node, ast.BoolLit):
        return expr_type(node)

    if isinstance(node, ast.Ident):
        return scope.lookup(node.name).type

    if isinstance(node, (ast.BinArith, ast.Compare,
                         ast.Logical, ast.Not, ast.Ruikei,
                         ast.CellField, ast.Jigen,
                         ast.ArrayLit, ast.CellLit)):
        # рекурсивно проверяем подузлы
        for child in node.__dict__.values():
            if isinstance(child, ast.Node):
                walk(child, scope)
            elif isinstance(child, list):
                for c in child:
                    if isinstance(c, ast.Node):
                        walk(c, scope)
        return expr_type(node)

    raise SemanticError(f"Неизвестный узел {node}")


# ------------------ вспомогательные ------------------------------------
def deduce_decl_type(vd: ast.VarDecl) -> str:
    if vd.dims is not None:
        return ARRAY
    if isinstance(vd.expr, ast.CellLit):
        return CELL
    if isinstance(vd.expr, ast.BoolLit):
        vd.is_const = True
        return BOOL
    if isinstance(vd.expr, ast.IntLit):
        vd.is_const = True
        return INT
    # по умолчанию int
    return INT


def require_assignable(dst: str, src: str, msg: str):
    """Допускаем bool→int и int→bool (0/1), всё остальное строгое."""
    if dst == src:
        return
    if dst == BOOL and src == INT:
        return
    raise SemanticError(msg)


# Backwards compatibility with parser expectations
semantic_check = check
