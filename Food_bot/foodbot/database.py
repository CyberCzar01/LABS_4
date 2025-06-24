"""Инфраструктурный слой: подключение к базе данных."""

from contextlib import asynccontextmanager

from sqlmodel import SQLModel
from sqlalchemy.ext.asyncio import AsyncEngine, create_async_engine
from sqlalchemy.orm import sessionmaker
from sqlmodel.ext.asyncio.session import AsyncSession
from sqlalchemy import text

from foodbot.config import settings

__all__ = ["get_session", "init_db", "engine"]

engine: AsyncEngine = create_async_engine(settings.database_url, echo=False, future=True)

async_session_factory = sessionmaker(  # type: ignore[call-arg]
    engine, expire_on_commit=False, class_=AsyncSession
)

if not hasattr(AsyncSession, "exec"):

    async def _exec(self, statement):  # type: ignore[override]
        """Имитация метода SQLModel.Session.exec для AsyncSession."""
        result = await self.execute(statement)
        return result.scalars()

    setattr(AsyncSession, "exec", _exec)

@asynccontextmanager
async def get_session():
    """Асинхронный контекст-менеджер для выдачи сессии БД."""
    async with async_session_factory() as session:  # noqa: WPS501
        yield session


async def init_db() -> None:
    """Создаёт все таблицы (если их нет). Для dev/стадии MVP."""
    async with engine.begin() as conn:
        await conn.run_sync(SQLModel.metadata.create_all)

        def _apply_migrations(sync_conn):
            # meal.canteen
            res = sync_conn.execute(text("PRAGMA table_info(meal);"))
            columns = [row[1] for row in res]
            if "canteen_id" not in columns:
                print("[migrate] Добавляю колонку meal.canteen_id …")
                sync_conn.execute(text("ALTER TABLE meal ADD COLUMN canteen_id INTEGER;"))

            # meal.description
            if "description" not in columns:
                print("[migrate] Добавляю колонку meal.description …")
                sync_conn.execute(text("ALTER TABLE meal ADD COLUMN description TEXT;"))

            # meal.is_complex
            if "is_complex" not in columns:
                print("[migrate] Добавляю колонку meal.is_complex …")
                sync_conn.execute(text("ALTER TABLE meal ADD COLUMN is_complex BOOLEAN DEFAULT 0;"))

            # order.menu_id / order.is_final — проверяем таблицу order
            res = sync_conn.execute(text("PRAGMA table_info(\"order\");"))
            order_columns = [row[1] for row in res]
            if "menu_id" not in order_columns:
                print("[migrate] Добавляю колонку order.menu_id …")
                sync_conn.execute(text("ALTER TABLE \"order\" ADD COLUMN menu_id INTEGER;"))
            if "is_final" not in order_columns:
                print("[migrate] Добавляю колонку order.is_final …")
                sync_conn.execute(text("ALTER TABLE \"order\" ADD COLUMN is_final BOOLEAN DEFAULT 0;"))

            # daily_menu.is_primary
            res = sync_conn.execute(text("PRAGMA table_info(dailymenu);"))
            dm_columns = [row[1] for row in res]
            if "is_primary" not in dm_columns:
                print("[migrate] Добавляю колонку dailymenu.is_primary …")
                sync_conn.execute(text("ALTER TABLE dailymenu ADD COLUMN is_primary BOOLEAN DEFAULT 0;"))

            if "parent_id" not in dm_columns:
                print("[migrate] Добавляю колонку dailymenu.parent_id …")
                sync_conn.execute(text("ALTER TABLE dailymenu ADD COLUMN parent_id INTEGER;"))

        await conn.run_sync(_apply_migrations) 