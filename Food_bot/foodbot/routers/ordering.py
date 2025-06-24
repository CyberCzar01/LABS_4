from __future__ import annotations

"""Роутер пользовательского меню и оформления заказа."""

from aiogram import Router, types, F
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from sqlmodel import select
from sqlalchemy import delete as sa_delete
from datetime import datetime, timedelta

from foodbot.database import get_session
from foodbot.models import User, Canteen, Meal, Order

menu_router = Router()


# Вспомогательные


async def _is_user_approved(tg_id: int) -> bool:
    """Проверяет, подтверждён ли пользователь."""
    async with get_session() as session:
        result = await session.exec(select(User).where(User.tg_id == tg_id))
        user = result.first()
        return bool(user and (user.is_approved or user.is_admin))


# Состояния FSM


class OrderStates(StatesGroup):
    waiting_canteen = State()
    waiting_meal = State()


# Старт оформления заказа


@menu_router.callback_query(lambda c: c.data == "order_start")
async def callback_order_start(call: types.CallbackQuery, state: FSMContext):
    if not await _is_user_approved(call.from_user.id):
        await call.answer("Сначала зарегистрируйтесь", show_alert=True)
        return

    # Сброс на случай повторного использования
    await state.clear()

    # Получаем активные столовые
    async with get_session() as session:
        result = await session.exec(select(Canteen).where(Canteen.is_active == True))  # noqa: E712
        canteens = list(result)

    if not canteens:
        await call.message.answer("Пока нет активных столовых 🙁")
        await call.answer()
        return

    kb_rows = [
        [types.InlineKeyboardButton(text=c.title, callback_data=f"canteen_select:{c.id}")] for c in canteens
    ]
    kb_rows.append([types.InlineKeyboardButton(text="❌ Отмена", callback_data="cancel")])
    kb = types.InlineKeyboardMarkup(inline_keyboard=kb_rows)

    await call.message.answer("Выберите столовую:", reply_markup=kb)
    await state.set_state(OrderStates.waiting_canteen)
    await call.answer()


# Выбор столовой


@menu_router.callback_query(OrderStates.waiting_canteen, lambda c: c.data and c.data.startswith("canteen_select:"))
async def callback_select_canteen(call: types.CallbackQuery, state: FSMContext):
    _, canteen_id_str = call.data.split(":", 1)
    canteen_id = int(canteen_id_str)

    async with get_session() as session:
        canteen = await session.get(Canteen, canteen_id)
        if not canteen or not canteen.is_active:
            await call.answer("Столовая недоступна", show_alert=True)
            return

        meals_result = await session.exec(select(Meal).where(Meal.canteen_id == canteen_id, Meal.is_active == True))  # noqa: E712
        meals = list(meals_result)

    if not meals:
        await call.answer("Меню пока пусто", show_alert=True)
        return

    await state.update_data(canteen_id=canteen_id)

    kb_rows = [
        [types.InlineKeyboardButton(text=m.title, callback_data=f"meal_select:{m.id}")] for m in meals
    ]
    kb_rows.append([types.InlineKeyboardButton(text="⬅️ Назад", callback_data="order_start")])
    kb = types.InlineKeyboardMarkup(inline_keyboard=kb_rows)

    await call.message.answer(f"Меню «{canteen.title}»:", reply_markup=kb)
    await state.set_state(OrderStates.waiting_meal)
    await call.answer()


# Выбор блюда


@menu_router.callback_query(OrderStates.waiting_meal, lambda c: c.data and c.data.startswith("meal_select:"))
async def callback_select_meal(call: types.CallbackQuery, state: FSMContext):
    _, meal_id_str = call.data.split(":", 1)
    meal_id = int(meal_id_str)

    # Проверим блюдо и создадим заказ
    async with get_session() as session:
        meal = await session.get(Meal, meal_id)
        if not meal or not meal.is_active:
            await call.answer("Блюдо недоступно", show_alert=True)
            return

        # находим пользователя по tg_id, берём его первичный ключ
        result = await session.exec(select(User).where(User.tg_id == call.from_user.id))
        db_user = result.first()
        if not db_user:
            await call.answer("Пользователь не найден", show_alert=True)
            return

        # Удаляем предыдущие заказы пользователя за эти сутки
        start_dt = datetime.utcnow().replace(hour=0, minute=0, second=0, microsecond=0)
        end_dt = start_dt + timedelta(days=1)

        await session.exec(
            sa_delete(Order).where(
                Order.user_id == db_user.id,
                Order.created_at >= start_dt,
                Order.created_at < end_dt,
            )
        )

        # Сохраняем новый заказ
        session.add(Order(user_id=db_user.id, meal_id=meal_id))
        await session.commit()

    await call.message.answer(f"✅ Заказ на «{meal.title}» принят! Приятного аппетита.",
                               reply_markup=types.InlineKeyboardMarkup(
                                   inline_keyboard=[[types.InlineKeyboardButton(text="🍴 Сделать ещё заказ", callback_data="order_start")]]
                               ))
    await state.clear()
    await call.answer()


#  Отмена


@menu_router.callback_query(lambda c: c.data == "cancel")
async def callback_cancel(call: types.CallbackQuery, state: FSMContext):
    await state.clear()
    await call.answer("Отменено")
    await call.message.answer("Если понадобится, нажмите /start чтобы начать заново.") 