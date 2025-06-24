from __future__ import annotations

from aiogram import Router, types
from sqlmodel import select
from sqlalchemy import delete as sa_delete
from datetime import datetime, timezone
from sqlalchemy.orm import selectinload

from foodbot.database import get_session
from foodbot.models import DailyMenu, MenuItem, Order, User, Meal
from foodbot.routers.registration import _get_admin_ids  # локальный импорт, чтобы не плодить код

menu_btn_router = Router()


@menu_btn_router.callback_query(lambda c: c.data == "menu_cancel")
async def cb_menu_cancel(call: types.CallbackQuery):
    await _cancel_order(call)


@menu_btn_router.callback_query(lambda c: c.data and c.data.startswith("menu:"))
async def menu_choice(call: types.CallbackQuery):
    # callback_data = "menu:<idx>"
    try:
        _, idx_str = call.data.split(":", 1)
        idx = int(idx_str)
    except Exception:  # noqa: BLE001
        await call.answer("Неверная кнопка", show_alert=True)
        return

    menu = await _get_active_menu(call.message.chat.id)
    if not menu:
        await call.answer("Меню недоступно", show_alert=True)
        return

    await _make_order(call, idx, menu)


async def _get_active_menu(chat_id: int):
    now = datetime.utcnow().replace(tzinfo=timezone.utc)
    async with get_session() as session:
        res = await session.exec(
            select(DailyMenu).where(
                DailyMenu.chat_id == chat_id,
                DailyMenu.is_closed == False,  # noqa: E712
                DailyMenu.deadline > now,
            )
        )
        return res.first()


async def _make_order(call: types.CallbackQuery, idx: int, menu):
    async with get_session() as session:
        # Если пользователь уже подтвердил заказ – правки запрещены
        user_res_chk = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        dbu_chk = user_res_chk.first()
        if dbu_chk:
            final_exists_res = await session.exec(
                select(Order).where(Order.user_id == dbu_chk.id, Order.menu_id == menu.id, Order.is_final == True)  # noqa: E712
            )
            if final_exists_res.first():
                await call.answer("Заказ уже подтверждён", show_alert=True)
                return

        item_res = await session.exec(
            select(MenuItem).where(MenuItem.menu_id == menu.id, MenuItem.button_index == idx)
        )
        item = item_res.first()
        if not item:
            await call.answer("Блюдо не найдено", show_alert=True)
            return

        user_res = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        db_user = user_res.first()
        if not db_user or not (db_user.is_approved or db_user.is_admin):
            await call.answer("Сначала зарегистрируйтесь", show_alert=True)
            return

        # Проверяем, есть ли уже такой заказ – тогглим
        existing_res = await session.exec(
            select(Order).where(
                Order.user_id == db_user.id,
                Order.menu_id == menu.id,
                Order.meal_id == item.meal_id,
            )
        )
        existing = existing_res.first()
        if existing:
            await session.delete(existing)
            await session.commit()
            await call.answer("❌ Блюдо убрано из заказа")
            return

        # Добавляем
        session.add(Order(user_id=db_user.id, meal_id=item.meal_id, menu_id=menu.id))
        await session.commit()

    await call.answer("✅ Добавлено к заказу")


async def _cancel_order(call: types.CallbackQuery):
    async with get_session() as session:
        menu = await _get_active_menu(call.message.chat.id)
        if not menu:
            await call.answer("Меню недоступно", show_alert=True)
            return

        user_res = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        db_user = user_res.first()
        if not db_user:
            await call.answer("Нет заказа", show_alert=True)
            return

        # Проверяем финальный статус
        final_orders_res = await session.exec(
            select(Order).where(Order.user_id == db_user.id, Order.menu_id == menu.id, Order.is_final == True)  # noqa: E712
        )
        if final_orders_res.first():
            await call.answer("Заказ уже подтверждён — изменения невозможны", show_alert=True)
            return

        all_item_res = await session.exec(select(MenuItem.meal_id).where(MenuItem.menu_id == menu.id))
        meal_ids_in_menu = list(all_item_res)
        await session.exec(sa_delete(Order).where(Order.user_id == db_user.id, Order.meal_id.in_(meal_ids_in_menu)))
        await session.commit()
    await call.answer("❌ Заказ удалён")


# Показываем описание


@menu_btn_router.callback_query(lambda c: c.data and c.data.startswith("menuinfo:"))
async def menu_info(call: types.CallbackQuery):
    try:
        _, idx_str = call.data.split(":", 1)
        idx = int(idx_str)
    except Exception:
        await call.answer("Ошибка", show_alert=True)
        return

    menu = await _get_active_menu(call.message.chat.id)
    if not menu:
        await call.answer("Меню недоступно", show_alert=True)
        return

    async with get_session() as session:
        item_res = await session.exec(
            select(MenuItem).where(MenuItem.menu_id == menu.id, MenuItem.button_index == idx)
        )
        item = item_res.first()
        if not item:
            await call.answer("Блюдо не найдено", show_alert=True)
            return

        meal = await session.get(Meal, item.meal_id)

    desc = meal.description or "Описание отсутствует"
    await call.answer(desc, show_alert=True)


# Нажали "Сделать заказ" — показываем предпросмотр


@menu_btn_router.callback_query(lambda c: c.data == "menu_submit")
async def cb_menu_preview(call: types.CallbackQuery):
    menu = await _get_active_menu(call.message.chat.id)
    if not menu:
        await call.answer("Меню недоступно", show_alert=True)
        return

    async with get_session() as session:
        user_res = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        db_user = user_res.first()
        if not db_user:
            await call.answer("Сначала зарегистрируйтесь", show_alert=True)
            return

        orders_res = await session.exec(
            select(Order).where(Order.user_id == db_user.id, Order.menu_id == menu.id)
        )
        user_orders = list(orders_res)

        if not user_orders:
            await call.answer("Вы ничего не выбрали", show_alert=True)
            return

        if any(o.is_final for o in user_orders):
            await call.answer("Вы уже подтвердили заказ", show_alert=True)
            return

        meals_res = await session.exec(select(Meal).where(Meal.id.in_([o.meal_id for o in user_orders])).options(selectinload(Meal.canteen)))
        id2title = {m.id: m.title for m in meals_res}

    lines = ["Ваш заказ:"] + [f"• {id2title[o.meal_id]}" for o in user_orders]
    preview_text = "\n".join(lines)

    kb = types.InlineKeyboardMarkup(
        inline_keyboard=[
            [types.InlineKeyboardButton(text="✅ Подтвердить", callback_data=f"menu_confirm:{menu.id}")],
            [types.InlineKeyboardButton(text="⬅️ Вернуться", callback_data="menu_back")],
        ]
    )
    await call.message.answer(preview_text, reply_markup=kb)
    await call.answer()


#  Подтверждение


@menu_btn_router.callback_query(lambda c: c.data and c.data.startswith("menu_confirm:"))
async def cb_menu_confirm(call: types.CallbackQuery):
    _, menu_id_str = call.data.split(":", 1)
    menu_id = int(menu_id_str)

    async with get_session() as session:
        menu = await session.get(DailyMenu, menu_id)
        if not menu or menu.is_closed:
            await call.answer("Меню недоступно", show_alert=True)
            return

        user_res = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        db_user = user_res.first()
        if not db_user:
            await call.answer("Сначала зарегистрируйтесь", show_alert=True)
            return

        orders_res = await session.exec(
            select(Order).where(Order.user_id == db_user.id, Order.menu_id == menu.id)
        )
        user_orders = list(orders_res)
        if not user_orders:
            await call.answer("Нет заказов", show_alert=True)
            return

        if any(o.is_final for o in user_orders):
            await call.answer("Уже подтверждено", show_alert=True)
            return

        for o in user_orders:
            o.is_final = True
        await session.commit()

    # убираем клавиатуру меню
    try:
        await call.message.edit_reply_markup(reply_markup=None)
    except Exception:
        pass

    await call.answer("✅ Заказ подтверждён! Изменить его теперь нельзя.", show_alert=True)


#  Вернуться к выбору


@menu_btn_router.callback_query(lambda c: c.data == "menu_back")
async def cb_menu_back(call: types.CallbackQuery):
    await call.answer("Можно продолжить выбор блюд")
    try:
        await call.message.delete()
    except Exception:
        pass


# Выбор столовой


@menu_btn_router.callback_query(lambda c: c.data and c.data.startswith("menu_can:"))
async def cb_menu_choose_canteen(call: types.CallbackQuery):
    """Показывает блюда выбранной столовой."""
    try:
        _, cid_str = call.data.split(":", 1)
        cid = int(cid_str)
    except Exception:
        await call.answer("Ошибка", show_alert=True)
        return

    menu = await _get_active_menu(call.message.chat.id)
    if not menu:
        await call.answer("Меню недоступно", show_alert=True)
        return

    async with get_session() as session:
        # Получаем все пункты меню и связанные блюда
        items_res = await session.exec(select(MenuItem).where(MenuItem.menu_id == menu.id))
        items = list(items_res)

        meal_ids = [it.meal_id for it in items]
        meals_res = await session.exec(select(Meal).where(Meal.id.in_(meal_ids)).options(selectinload(Meal.canteen)))
        id2meal = {m.id: m for m in meals_res}

    # Отбираем блюда выбранной столовой
    filtered: list[tuple[int, Meal]] = []  # (button_idx, meal)
    for it in items:
        meal = id2meal.get(it.meal_id)
        if not meal:
            continue
        if (meal.canteen_id or 0) == cid:
            filtered.append((it.button_index, meal))

    # Получаем название столовой
    canteen_title = "Столовая"
    if filtered:
        m0 = filtered[0][1]
        if m0.canteen:
            canteen_title = m0.canteen.title

    if not filtered:
        await call.answer("Меню пусто", show_alert=True)
        return

    # Формируем текстовый список блюд. Комплексные обеды — с вложенными строками описания.
    text_lines = [f"<b>{canteen_title}</b>"]
    for _, meal in filtered:
        if meal.is_complex:
            text_lines.append(f"🍱 {meal.title}")
            if meal.description:
                for line in meal.description.splitlines():
                    line = line.strip()
                    if line:
                        nbsp = "\u00A0" * 2  # два неразрывных пробела, телега их показывает
                        text_lines.append(f"{nbsp}• {line}")
        else:
            text_lines.append(f"• {meal.title}")

    new_text = "\n".join(text_lines)

    # Строим клавиатуру: блюдо | info
    kb_rows: list[list[types.InlineKeyboardButton]] = []
    for idx, meal in filtered:
        kb_rows.append([
            types.InlineKeyboardButton(text=meal.title, callback_data=f"menu:{idx}"),
            types.InlineKeyboardButton(text="ℹ️", callback_data=f"menuinfo:{idx}"),
        ])

    # Навигация назад к списку столовых
    kb_rows.append([
        types.InlineKeyboardButton(text="⬅️ Назад", callback_data="menu_cans"),
    ])
    kb_rows.append([
        types.InlineKeyboardButton(text="🤷‍♂️ Отменить выбор", callback_data="menu_cancel"),
    ])
    kb_rows.append([
        types.InlineKeyboardButton(text="✅ Сделать заказ", callback_data="menu_submit"),
    ])

    # Изменяем текст и клавиатуру
    try:
        await call.message.edit_text(new_text, reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows), parse_mode="HTML")
    except Exception:
        # если изменять нельзя, отправляем новое
        await call.message.answer(new_text, reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows), parse_mode="HTML")
    await call.answer()


@menu_btn_router.callback_query(lambda c: c.data == "menu_cans")
async def cb_menu_back_to_canteens(call: types.CallbackQuery):
    """Возврат к списку столовых в меню дня."""
    menu = await _get_active_menu(call.message.chat.id)
    if not menu:
        await call.answer("Меню недоступно", show_alert=True)
        return

    async with get_session() as session:
        items_res = await session.exec(select(MenuItem).where(MenuItem.menu_id == menu.id))
        items = list(items_res)
        meal_ids = [it.meal_id for it in items]
        meals_res = await session.exec(select(Meal).where(Meal.id.in_(meal_ids)).options(selectinload(Meal.canteen)))
        id2meal = {m.id: m for m in meals_res}

    canteen_ids = {id2meal[it.meal_id].canteen_id or 0 for it in items}
    id2title: dict[int, str] = {}
    for meal in id2meal.values():
        cid = meal.canteen_id or 0
        if cid not in id2title:
            id2title[cid] = meal.canteen.title if meal.canteen else "Прочее"

    kb_rows: list[list[types.InlineKeyboardButton]] = []
    for cid in sorted(canteen_ids):
        kb_rows.append([
            types.InlineKeyboardButton(text=id2title[cid], callback_data=f"menu_can:{cid}"),
        ])

    kb_rows.append([
        types.InlineKeyboardButton(text="🤷‍♂️ Отменить выбор", callback_data="menu_cancel"),
    ])
    kb_rows.append([
        types.InlineKeyboardButton(text="✅ Сделать заказ", callback_data="menu_submit"),
    ])

    new_text = "Выберите столовую:"  # текст для верхнего сообщения

    try:
        await call.message.edit_text(new_text, reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows))
    except Exception:
        await call.message.answer(new_text, reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows))
    await call.answer() 