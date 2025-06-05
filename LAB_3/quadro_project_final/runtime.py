"""
quadro.runtime – рантайм Quadro: представления значений, среда,
функции, создание массивов и т.д.
"""

from __future__ import annotations
from typing import Any, List

# --------------------- Базовый класс для значений ------------------------
class Value:
    def as_int(self) -> int:
        raise NotImplementedError
    def as_bool(self) -> bool:
        raise NotImplementedError

# ----------------------------------------------------------------------
#  Целое число
# ----------------------------------------------------------------------
class IntVal(Value):
    def __init__(self, v: int):
        self.v = v
    def as_int(self) -> int:
        return self.v
    def as_bool(self) -> bool:
        return self.v != 0
    def __repr__(self):
        return f"IntVal({self.v})"

# ----------------------------------------------------------------------
#  Логическое (bool)
# ----------------------------------------------------------------------
class BoolVal(Value):
    def __init__(self, b: bool):
        self.b = b
    def as_int(self) -> int:
        return 1 if self.b else 0
    def as_bool(self) -> bool:
        return self.b
    def __repr__(self):
        return f"BoolVal({self.b})"

# ----------------------------------------------------------------------
#  Ячейка (CellVal)
# ----------------------------------------------------------------------
class CellVal(Value):
    def __init__(self, x: int, y: int, z: int, busy: bool):
        self.x = x; self.y = y; self.z = z; self.busy = busy
    def as_int(self) -> int:
        # Обычно int-представление не используется
        return 0
    def as_bool(self) -> bool:
        # нельзя приводить cell к bool прямо
        raise RuntimeError("CellVal cannot be cast to bool")
    def __repr__(self):
        return f"CellVal(x={self.x},y={self.y},z={self.z},busy={self.busy})"

# ----------------------------------------------------------------------
#  Массив (ArrayVal) – поддерживаем N-мерный массив через nested lists
# ----------------------------------------------------------------------
class ArrayVal(Value):
    def __init__(self, dims: List[int] | List[Value] | None = None):
        """
        Если dims – список int, создаём вложенный список размеров dims, заполненный IntVal(0).
        Если dims – список Value-объектов (т.е. при literal), сохраняем эти объекты в data.
        """
        if dims is None:
            # корень одномерного массива по умолчанию пустой
            self.dims: List[int] = []
            self.data: List[Value] = []
        else:
            if isinstance(dims[0], Value):
                # если это список значений (например, из Sequence)
                self.data = dims
                self.dims = []  # dims может быть несущественным
            else:
                # dims – список целых чисел: создаём nested lists
                self.dims = [int(d) for d in dims]
                self.data = self._create_nested(self.dims.copy(), IntVal(0))

    @staticmethod
    def _create_nested(dims: List[int], fill: Value) -> Any:
        if not dims:
            return fill
        size = dims.pop(0)
        return [ArrayVal._create_nested(dims.copy(), fill) for _ in range(size)]

    def as_int(self) -> int:
        raise TypeError("array → int")

    def as_bool(self) -> bool:
        raise TypeError("array → bool")

    def __repr__(self):
        return f"ArrayVal(dims={self.dims}, data={self.data})"


# ----------------------------------------------------------------------
#  Окружение (словари имён → значения)
# ----------------------------------------------------------------------
class Environment:
    def __init__(self, parent: Environment | None = None):
        self.vars: dict[str, Value] = {}
        self.parent = parent

    def __setitem__(self, key: str, value: Value):
        self.vars[key] = value

    def __getitem__(self, key: str) -> Value:
        return self.get(key)

    def get(self, key: str) -> Value:
        if key in self.vars:
            return self.vars[key]
        if self.parent:
            return self.parent.get(key)
        raise RuntimeError(f"Variable {key!r} not found")


# ----------------------------------------------------------------------
#  Функции (применяются в FuncCall)
# ----------------------------------------------------------------------
class FunctionVal(Value):
    def __init__(self, params: List[str], body: "Block", defining_env: "Environment"):
        self.param_names = params
        self.body = body
        self.def_env = defining_env

    def call(self, args: List[Value], caller_env: Environment) -> Value:
        if len(args) != len(self.param_names):
            raise TypeError("Неверное число аргументов")
        # создаём окружение для функции
        local_env = Environment(parent=self.def_env)
        for name, val in zip(self.param_names, args):
            local_env[name] = val
        # выполняем тело
        self.body.eval(local_env)
        # по ТЗ функции не возвращают значение (или Int по умолчанию)
        return IntVal(0)

    def as_int(self) -> int:
        raise RuntimeError("Function cannot be cast to int")

    def as_bool(self) -> bool:
        raise RuntimeError("Function cannot be cast to bool")


# ----------------------------------------------------------------------
#  Robot and maze utilities
# ----------------------------------------------------------------------
MOVE_DIR = {
    '^_^': (0, -1, 0),  # up
    'v_v': (0, 1, 0),   # down
    '<_<': (-1, 0, 0),  # left
    '>_>': (1, 0, 0),   # right
    'o_o': (0, 0, -1),  # forward (z-)
    '~_~': (0, 0, 1),   # back (z+)
}

MEAS_DIR = {
    '^_0': (0, -1, 0),
    'v_0': (0, 1, 0),
    '<_0': (-1, 0, 0),
    '>_0': (1, 0, 0),
    'o_0': (0, 0, -1),
    '~_0': (0, 0, 1),
}


class Robot:
    def __init__(self, maze: list[list[list[str]]], start: tuple[int, int, int], exits: list[tuple[int, int, int]]):
        self.maze = maze
        self.width = len(maze[0][0])
        self.height = len(maze[0])
        self.depth = len(maze)
        self.pos = list(start)
        self.exits = exits
        self.crashed = False

    # helpers
    def _in_bounds(self, x: int, y: int, z: int) -> bool:
        return 0 <= x < self.width and 0 <= y < self.height and 0 <= z < self.depth

    def _is_free(self, x: int, y: int, z: int) -> bool:
        return self._in_bounds(x, y, z) and self.maze[z][y][x] == '0'

    def move(self, token: str) -> None:
        if self.crashed:
            return
        dx, dy, dz = MOVE_DIR[token]
        nx, ny, nz = self.pos[0] + dx, self.pos[1] + dy, self.pos[2] + dz
        if not self._is_free(nx, ny, nz):
            self.crashed = True
        else:
            self.pos = [nx, ny, nz]

    def measure(self, token: str) -> int:
        dx, dy, dz = MEAS_DIR[token]
        dist = 0
        x, y, z = self.pos
        while True:
            x += dx; y += dy; z += dz
            if not self._in_bounds(x, y, z) or self.maze[z][y][x] == '1':
                break
            dist += 1
        return dist

    def get_pos(self) -> CellVal:
        return CellVal(self.pos[0], self.pos[1], self.pos[2], False)


def load_labyrinth(path: str):
    lines = [line.strip() for line in open(path, encoding='utf-8').read().splitlines() if line.strip()]
    w, h, d = map(int, lines[0].split())
    idx = 1
    maze = []
    for _ in range(d):
        layer = []
        for _ in range(h):
            layer.append(list(lines[idx].strip()))
            idx += 1
        maze.append(layer)
    exits_count = int(lines[idx]); idx += 1
    exits = []
    for _ in range(exits_count):
        x, y, z = map(int, lines[idx].split())
        exits.append((x, y, z))
        idx += 1
    start = tuple(map(int, lines[idx].split()))
    return maze, exits, start


_current_robot: Robot | None = None


def runtime_init(maze_file: str) -> None:
    global _current_robot
    maze, exits, start = load_labyrinth(maze_file)
    _current_robot = Robot(maze, start, exits)


def robot_move(direction: str) -> None:
    if _current_robot:
        _current_robot.move(direction)


def robot_measure(direction: str) -> int:
    if _current_robot:
        return _current_robot.measure(direction)
    return 0


def robot_get_pos() -> CellVal:
    if _current_robot:
        return _current_robot.get_pos()
    return CellVal(0, 0, 0, False)


def run_program(program, maze_file: str) -> None:
    runtime_init(maze_file)
    env = Environment()
    env['__robot__'] = _current_robot
    env['_last_meas'] = IntVal(0)
    program.eval(env)
