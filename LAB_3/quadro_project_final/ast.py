"""
quadro.ast – полный AST для Quadro (Stage 3-a / блок C-D)
Совместимо с Python 3.9, включает семантику, Sequence, функции, робо-команды и т.д.
"""
from __future__ import annotations

from dataclasses import dataclass
from typing import List, Optional, Any, Iterator

from .runtime import (
    IntVal, BoolVal, CellVal, ArrayVal,
    Environment, FunctionVal
)
from .robot import Robot, MEAS_DIR
from .semantic import SemanticError


# --------------------- Базовый класс всех узлов -----------------------
class Node:
    def eval(self, env: Environment):
        raise NotImplementedError


# ----------------------------------------------------------------------
#  Программа и блок
# ----------------------------------------------------------------------
@dataclass
class Program(Node):
    body: List[Node]
    def eval(self, env):
        for st in self.body:
            st.eval(env)

@dataclass
class Block(Node):
    stmts: List[Node]
    def eval(self, env):
        # новая область
        block_env = Environment(parent=env)
        for st in self.stmts:
            st.eval(block_env)


# ----------------------------------------------------------------------
#  Объявление переменной
# ----------------------------------------------------------------------
@dataclass
class VarDecl(Node):
    name: str
    expr: Optional[Node] = None      # None ⇒ 0 / uso
    dims: Optional[List[Any]] = None # либо list[int], либо list[Node]
    is_const: bool = False

    def eval(self, env: Environment):
        if self.expr:
            env[self.name] = self.expr.eval(env)
        elif self.dims is not None:
            # Если dims – список узлов AST, вычисляем их сейчас:
            if all(hasattr(d, 'eval') for d in self.dims):
                evaluated = [d.eval(env).as_int() for d in self.dims]
            else:
                evaluated = self.dims  # уже были ints
            env[self.name] = ArrayVal(dims=evaluated)
        else:
            env[self.name] = IntVal(0)


# ----------------------------------------------------------------------
#  Присваивание
# ----------------------------------------------------------------------
@dataclass
class Assign(Node):
    target: str
    expr: Node
    def eval(self, env: Environment):
        env[self.target] = self.expr.eval(env)


# ----------------------------------------------------------------------
#  Литералы
# ----------------------------------------------------------------------
@dataclass
class IntLit(Node):
    value: int
    def eval(self, env: Environment):
        return IntVal(self.value)

@dataclass
class BoolLit(Node):
    value: bool
    def eval(self, env: Environment):
        return BoolVal(self.value)


# ----------------------------------------------------------------------
#  Идентификатор
# ----------------------------------------------------------------------
@dataclass
class Ident(Node):
    name: str
    def eval(self, env: Environment):
        return env.get(self.name)


# ----------------------------------------------------------------------
#  Бинарная арифметика (+, -, *)
# ----------------------------------------------------------------------
@dataclass
class BinArith(Node):
    op: str
    left: Node
    right: Node

    def eval(self, env: Environment):
        lval = self.left.eval(env).as_int()
        rval = self.right.eval(env).as_int()
        if self.op == '+':
            return IntVal(lval + rval)
        if self.op == '-':
            return IntVal(lval - rval)
        if self.op == '*':
            return IntVal(lval * rval)
        raise RuntimeError(f"Unknown arithmetic op {self.op!r}")


# ----------------------------------------------------------------------
#  Сравнение (<, >)
# ----------------------------------------------------------------------
@dataclass
class Compare(Node):
    op: str
    left: Node
    right: Node
    def eval(self, env: Environment):
        lval = self.left.eval(env).as_int()
        rval = self.right.eval(env).as_int()
        if self.op == '<':
            return BoolVal(lval < rval)
        if self.op == '>':
            return BoolVal(lval > rval)
        raise RuntimeError(f"Unknown compare op {self.op!r}")


# ----------------------------------------------------------------------
#  Логика (~, ^, v)
# ----------------------------------------------------------------------
@dataclass
class Not(Node):
    expr: Node
    def eval(self, env: Environment):
        ev = self.expr.eval(env)
        b = ev.as_bool()
        return BoolVal(not b)

@dataclass
class Logical(Node):
    op: str  # '^' или 'v'
    left: Node
    right: Node
    def eval(self, env: Environment):
        b1 = self.left.eval(env).as_bool()
        b2 = self.right.eval(env).as_bool()
        if self.op == '^':
            return BoolVal(b1 and b2)
        if self.op == 'v':
            return BoolVal(b1 or b2)
        raise RuntimeError(f"Unknown logical op {self.op!r}")


# ----------------------------------------------------------------------
#  ruikei (определение типа)
# ----------------------------------------------------------------------
@dataclass
class Ruikei(Node):
    left: Node
    right: Node
    def eval(self, env: Environment):
        # Не нужна реальная оценка, это только для семантики
        return BoolVal(expr_type(self.left) == expr_type(self.right))


# ----------------------------------------------------------------------
#  rippotai => поле (x,y,z либо busy)
# ----------------------------------------------------------------------
@dataclass
class CellField(Node):
    expr: Node
    field: str  # 'x','y','z' или 'busy'
    def eval(self, env: Environment):
        cell: CellVal = self.expr.eval(env)
        if self.field == 'x':
            return IntVal(cell.x)
        if self.field == 'y':
            return IntVal(cell.y)
        if self.field == 'z':
            return IntVal(cell.z)
        if self.field == 'busy':
            return BoolVal(cell.busy)
        raise RuntimeError(f"Unknown cell field {self.field!r}")


# ----------------------------------------------------------------------
#  Jigen (размерность массива)
# ----------------------------------------------------------------------
@dataclass
class Jigen(Node):
    array_name: str
    def eval(self, env: Environment):
        arr: ArrayVal = env.get(self.array_name)
        return ArrayVal([IntVal(d) for d in arr.dims])


# ----------------------------------------------------------------------
#  Литеральный массив [ .... ]
# ----------------------------------------------------------------------
@dataclass
class ArrayLit(Node):
    elems: List[Node]
    def eval(self, env: Environment):
        values = [e.eval(env) for e in self.elems]
        # Считаем всё одномерным списком
        data = [v for v in values]
        return ArrayVal(data)


# ----------------------------------------------------------------------
#  Литеральная ячейка (rippotai с конкретными координатами и busy)
# ----------------------------------------------------------------------
@dataclass
class CellLit(Node):
    x: Node
    y: Node
    z: Node
    busy: Node
    def eval(self, env: Environment):
        xv = self.x.eval(env).as_int()
        yv = self.y.eval(env).as_int()
        zv = self.z.eval(env).as_int()
        bv = self.busy.eval(env).as_bool()
        return CellVal(xv, yv, zv, bv)


# ----------------------------------------------------------------------
#  robо-команды
# ----------------------------------------------------------------------
@dataclass
class MoveCmd(Node):
    token: str
    def eval(self, env: Environment):
        robot: Robot = env.get('__robot__')
        robot.move(self.token)
        return IntVal(0)

@dataclass
class MeasureCmd(Node):
    token: str
    def eval(self, env: Environment):
        robot: Robot = env.get('__robot__')
        dist = robot.measure(self.token)
        env['_last_meas'] = IntVal(dist)
        return IntVal(dist)

@dataclass
class StopIf(Node):
    def eval(self, env: Environment):
        # прерываем Sequence, если прошлое измерение == 1
        if env.get('_last_meas').as_int() == 1:
            raise RuntimeError('Sequence stopped')
        return IntVal(0)


# ----------------------------------------------------------------------
#  Sequence { ... }
# ----------------------------------------------------------------------
@dataclass
class Sequence(Node):
    cmds: List[Node]

    def eval(self, env: Environment):
        robot: Robot = env.get('__robot__')
        visited_abs: dict[tuple[int,int,int], CellVal] = {}

        for cmd in self.cmds:
            if isinstance(cmd, MeasureCmd):
                dist = cmd.eval(env).as_int()
                dx, dy, dz = MEAS_DIR[cmd.token]
                ox, oy, oz = robot.pos
                for step in range(1, dist + 1):
                    x, y, z = ox + dx*step, oy + dy*step, oz + dz*step
                    busy = (step == dist)
                    visited_abs[(x, y, z)] = CellVal(x, y, z, busy)

            elif isinstance(cmd, StopIf):
                if env.get('_last_meas').as_int() == 1:
                    break
            else:
                cmd.eval(env)

        if not visited_abs:
            return ArrayVal([])

        cx, cy, cz = robot.pos
        rel_cells = { (x-cx, y-cy, z-cz): cell
                      for (x,y,z), cell in visited_abs.items() }

        xs = [c[0] for c in rel_cells]
        ys = [c[1] for c in rel_cells]
        zs = [c[2] for c in rel_cells]
        minx, maxx = min(xs), max(xs)
        miny, maxy = min(ys), max(ys)
        minz, maxz = min(zs), max(zs)

        matrix_z: List[ArrayVal] = []
        for z in range(minz, maxz+1):
            layer_y: List[ArrayVal] = []
            for y in range(miny, maxy+1):
                row_x: List[CellVal] = []
                for x in range(minx, maxx+1):
                    row_x.append(rel_cells.get(
                        (x,y,z),
                        CellVal(x, y, z, False)))
                layer_y.append(ArrayVal(row_x))
            matrix_z.append(ArrayVal(layer_y))

        return ArrayVal(matrix_z)


# ----------------------------------------------------------------------
#  WhereExpr (*_*)
# ----------------------------------------------------------------------
@dataclass
class WhereExpr(Node):
    def eval(self, env: Environment):
        robot: Robot = env.get('__robot__')
        return CellVal(*robot.pos, False)


# ----------------------------------------------------------------------
#  If / For
# ----------------------------------------------------------------------
@dataclass
class If(Node):
    cond: Node
    body: Block
    def eval(self, env: Environment):
        if self.cond.eval(env).as_bool():
            self.body.eval(env)

@dataclass
class For(Node):
    var: str
    start: Node
    end: Node
    body: Block
    def eval(self, env: Environment):
        s = self.start.eval(env).as_int()
        e = self.end.eval(env).as_int()
        for i in range(s, e + 1):
            env_local = Environment(parent=env)
            env_local[self.var] = IntVal(i)
            self.body.eval(env_local)


# ----------------------------------------------------------------------
#  Функции kansu
# ----------------------------------------------------------------------
@dataclass
class FuncDef(Node):
    name: str
    param_names: List[str]
    body: Block
    def eval(self, env: Environment):
        fn = FunctionVal(self.param_names, self.body, env)
        env[self.name] = fn

@dataclass
class FuncCall(Node):
    name: str
    args: List[Node]

    def eval(self, env: Environment):
        fn: FunctionVal = env.get(self.name)
        arg_vals = [a.eval(env) for a in self.args]
        try:
            return fn.call(arg_vals, env)
        except TypeError as e:
            raise SemanticError(str(e))


# ----------------------------------------------------------------------
#  Помощник для expr_type в семантике
# ----------------------------------------------------------------------
def expr_type(node: Node) -> str:
    from .semantic import INT, BOOL, CELL, ARRAY, FUNC
    if isinstance(node, IntLit):        return INT
    if isinstance(node, BoolLit):       return BOOL
    if isinstance(node, CellLit):       return CELL
    if isinstance(node, ArrayLit):      return ARRAY
    if isinstance(node, Ident):         return 'ident'
    if isinstance(node, BinArith):      return INT
    if isinstance(node, Compare):       return BOOL
    if isinstance(node, Logical):       return BOOL
    if isinstance(node, Not):           return BOOL
    if isinstance(node, Ruikei):        return BOOL
    if isinstance(node, CellField):
        return INT if node.field in ('x','y','z') else BOOL
    if isinstance(node, Jigen):         return ARRAY
    if isinstance(node, FuncCall):      return INT
    if isinstance(node, WhereExpr):     return CELL
    if isinstance(node, Sequence):      return ARRAY
    return 'unknown'
