from datetime import datetime
from typing import Optional

from sqlmodel import Field, SQLModel, Relationship


# Вспомогательная функция для единообразного времени
def _now_utc() -> datetime:
    """Возвращает текущее время в UTC (naive)."""
    return datetime.utcnow()


class User(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    tg_id: int = Field(index=True, unique=True, nullable=False)
    full_name: str
    is_approved: bool = False
    is_admin: bool = False
    created_at: datetime = Field(default_factory=_now_utc)

    orders: list["Order"] = Relationship(back_populates="user")


class Canteen(SQLModel, table=True):
    """Столовая (точка питания)."""

    id: Optional[int] = Field(default=None, primary_key=True)
    title: str
    is_active: bool = True
    created_at: datetime = Field(default_factory=_now_utc)

    meals: list["Meal"] = Relationship(back_populates="canteen")


class Meal(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    title: str
    description: str | None = ""  # подробности блюда / состав комплексного обеда
    is_complex: bool = False  # комплексный обед?
    canteen_id: int | None = Field(default=None, foreign_key="canteen.id")
    is_active: bool = True
    created_at: datetime = Field(default_factory=_now_utc)

    orders: list["Order"] = Relationship(back_populates="meal")
    canteen: Canteen | None = Relationship(back_populates="meals")


class Order(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    user_id: int = Field(foreign_key="user.id")
    meal_id: int = Field(foreign_key="meal.id")  # косвенно содержит canteen
    menu_id: int | None = Field(default=None, foreign_key="dailymenu.id")
    is_final: bool = False  # пользователь подтвердил заказ?
    created_at: datetime = Field(default_factory=_now_utc)

    user: User = Relationship(back_populates="orders")
    meal: Meal = Relationship(back_populates="orders")


# ===== Меню дня (inline-кнопки) =====


class DailyMenu(SQLModel, table=True):
    """Сообщение-меню, опубликованное ботом на конкретную дату."""

    id: Optional[int] = Field(default=None, primary_key=True)
    message_id: int = Field(index=True)
    chat_id: int = Field(index=True)
    deadline: datetime  # UTC
    is_closed: bool = False
    is_primary: bool = False  # меню в админ-чате
    parent_id: int | None = Field(default=None, foreign_key="dailymenu.id")  # у копии ссылается на основное меню
    created_at: datetime = Field(default_factory=_now_utc)

    items: list["MenuItem"] = Relationship(back_populates="menu")


class MenuItem(SQLModel, table=True):
    id: Optional[int] = Field(default=None, primary_key=True)
    menu_id: int = Field(foreign_key="dailymenu.id")
    meal_id: int = Field(foreign_key="meal.id")
    button_index: int  # индекс кнопки

    menu: DailyMenu = Relationship(back_populates="items")
    meal: Meal = Relationship() 