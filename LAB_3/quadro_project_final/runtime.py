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
