from __future__ import annotations

import asyncio
from datetime import datetime, timezone
from typing import Sequence
from zoneinfo import ZoneInfo

from aiogram import Bot, types
from sqlmodel import select
from sqlalchemy.orm import selectinload

from foodbot.database import get_session
from foodbot.models import DailyMenu, MenuItem, Meal, Order, User
from foodbot.config import settings

__all__ = [
    "create_menu",
    "close_menu",
    "deadline_checker",
]


async def create_menu(*, bot: Bot, chat_id: int, deadline: datetime, meal_ids: Sequence[int], primary: bool = False, parent_id: int | None = None) -> DailyMenu:  # noqa: D401
    """Публикует сообщение-меню с inline-кнопками и записывает DailyMenu/Item."""
    # Загружаем блюда
    async with get_session() as session:
        stmt = select(Meal).where(Meal.id.in_(meal_ids)).options(selectinload(Meal.canteen))
        meals = list(await session.exec(stmt))
    if not meals:
        raise ValueError("no_meals")

    tz = settings.tz
    dl_local = deadline.astimezone(tz)
    text_lines = [
        f"🍽️ Заказ на {dl_local.strftime('%d.%m')} (до {dl_local.strftime('%H:%M')})",
        "",
    ]
    # Перечисляем блюда только если меню для одной столовой.
    if len({m.canteen.id if m.canteen else 0 for m in meals}) == 1:
        for idx, meal in enumerate(meals, start=1):
            prefix = ""
            if meal.canteen:
                prefix = f"{meal.canteen.title} — "

            # Для обычных блюд (не комплексных) добавляем описание в скобках, если оно есть,
            # чтобы различать позиции с одинаковым названием.
            title_part = meal.title
            if not meal.is_complex and meal.description:
                title_part = f"{meal.title} ({meal.description})"

            text_lines.append(f"{idx}. {prefix}{title_part}")
    text = "\n".join(text_lines)

    # === Формируем клавиатуру ===

    # Если блюда из нескольких столовых, показываем сначала список столовых,
    # иначе сразу список блюд.

    canteen_ids = {m.canteen.id if m.canteen else 0 for m in meals}

    def _manage_rows(rows: list[list[types.InlineKeyboardButton]]) -> list[list[types.InlineKeyboardButton]]:
        """Добавляет управляющие кнопки (отмена / отправка заказа)."""
        rows.append([types.InlineKeyboardButton(text="🤷‍♂️ Отменить выбор", callback_data="menu_cancel")])
        rows.append([types.InlineKeyboardButton(text="✅ Сделать заказ", callback_data="menu_submit")])
        return rows

    if len(canteen_ids) > 1:
        # Первым шагом пользователь выбирает столовую
        kb_rows: list[list[types.InlineKeyboardButton]] = []
        # Группируем блюда по столовым, чтобы подсчитать кол-во/получить название
        id2title: dict[int, str] = {}
        for m in meals:
            if m.canteen:
                id2title[m.canteen.id] = m.canteen.title
            else:
                id2title[0] = "Прочее"

        # Столовые в алфавитном порядке
        for cid, title in sorted(id2title.items(), key=lambda t: t[1].lower()):
            kb_rows.append([
                types.InlineKeyboardButton(text=title, callback_data=f"menu_can:{cid}")
            ])

        markup = types.InlineKeyboardMarkup(inline_keyboard=_manage_rows(kb_rows))
    else:
        # Одна столовая - показываем блюда напрямую, как раньше
        kb_rows: list[list[types.InlineKeyboardButton]] = []
        for idx, meal in enumerate(meals):
            label = meal.title
            if not meal.is_complex and meal.description:
                label = f"{meal.title} ({meal.description})"
                # Ограничим длину, чтобы кнопка не была слишком широкой
                if len(label) > 60:
                    label = label[:57] + "…"

            select_btn = types.InlineKeyboardButton(text=label, callback_data=f"menu:{idx}")
            info_btn = types.InlineKeyboardButton(text="ℹ️", callback_data=f"menuinfo:{idx}")
            kb_rows.append([select_btn, info_btn])
        markup = types.InlineKeyboardMarkup(inline_keyboard=_manage_rows(kb_rows))

    msg = await bot.send_message(chat_id, text, reply_markup=markup)

    async with get_session() as session:
        menu = DailyMenu(message_id=msg.message_id, chat_id=msg.chat.id, deadline=deadline.astimezone(timezone.utc), is_primary=primary, parent_id=parent_id)
        session.add(menu)
        await session.commit()
        await session.refresh(menu)
        for idx, meal in enumerate(meals):
            session.add(MenuItem(menu_id=menu.id, meal_id=meal.id, button_index=idx))
        await session.commit()
    return menu


async def _orders_for_menu(menu: DailyMenu):
    async with get_session() as session:
        items_res = await session.exec(select(MenuItem).where(MenuItem.menu_id == menu.id))
        meal_ids = [itm.meal_id for itm in items_res]
        orders_res = await session.exec(select(Order).where(Order.meal_id.in_(meal_ids), Order.is_final == True))  # noqa: E712
        return list(orders_res)


async def close_menu(bot: Bot, menu: DailyMenu) -> None:
    """Помечает меню как закрыто и убирает inline-кнопки."""
    if menu.is_closed:
        return

    # редактируем сообщение
    try:
        await bot.edit_message_reply_markup(menu.chat_id, menu.message_id, reply_markup=None)
        await bot.edit_message_text(
            chat_id=menu.chat_id,
            message_id=menu.message_id,
            text="❌ Приём заказов завершён\n" + ("\n".join((await _format_summary(menu)))),
        )
    except Exception:  # noqa: BLE001
        pass

    # флаг закрыт
    async with get_session() as session:
        db_menu = await session.get(DailyMenu, menu.id)
        if db_menu:
            db_menu.is_closed = True
            await session.commit()

    # после закрытия формируем отчёт только для первичного меню
    if not menu.is_primary:
        return

    try:
        from foodbot.services.orders import build_menu_report
        from foodbot.routers.registration import _get_admin_ids

        filename, buffer = await build_menu_report(menu.id)

        admin_ids = await _get_admin_ids(bot)
        for aid in admin_ids:
            try:
                buffer.seek(0)
                await bot.send_document(
                    aid,
                    types.BufferedInputFile(buffer.read(), filename=filename),
                    caption="📊 Отчёт — меню завершено",
                )
            except Exception:  # noqa: BLE001
                continue
    except ValueError:
        # заказов нет — ничего не отправляем
        pass


async def _format_summary(menu: DailyMenu) -> list[str]:
    orders = await _orders_for_menu(menu)
    summary: dict[int, int] = {}
    for order in orders:
        summary[order.meal_id] = summary.get(order.meal_id, 0) + 1

    if not summary:
        return ["Заказов нет"]

    async with get_session() as session:
        meals_res = await session.exec(select(Meal).where(Meal.id.in_(summary.keys())))
        id_to_title: dict[int, str] = {}
        for m in meals_res:
            title = m.title
            if not m.is_complex and m.description:
                title = f"{m.title} ({m.description})"
            id_to_title[m.id] = title

    return [f"• {id_to_title[mid]} — {cnt} шт." for mid, cnt in summary.items()]


async def deadline_checker(bot: Bot, *, interval_sec: int = 60) -> None:  # noqa: D401
    """Фоновая корутина: каждые N секунд закрывает просроченные меню."""
    while True:
        now = datetime.utcnow().replace(tzinfo=timezone.utc)
        async with get_session() as session:
            res = await session.exec(
                select(DailyMenu).where(DailyMenu.is_closed == False, DailyMenu.deadline <= now)  # noqa: E712
            )
            menus = list(res)
        for menu in menus:
            await close_menu(bot, menu)
        await asyncio.sleep(interval_sec) 