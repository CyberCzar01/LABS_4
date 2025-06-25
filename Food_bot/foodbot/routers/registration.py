"""Маршруты, связанные с регистрацией пользователя."""

from aiogram import Router, F, Bot, types
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup
from sqlmodel import select

from foodbot.config import settings
from foodbot.database import get_session
from foodbot.models import User

registration_router = Router()


class RegStates(StatesGroup):
    waiting_full_name = State()


async def _get_admin_ids(bot: Bot) -> list[int]:
    """Получить tg_id админов по username из .env (one-time lookup)."""
    admin_ids: set[int] = set()

    # 1. usernames из .env
    for raw_username in settings.admin_usernames.split() or []:
        raw_username = raw_username.strip()
        if not raw_username:
            continue
        try:
            chat = await bot.get_chat(raw_username)
            admin_ids.add(chat.id)
        except Exception:  # noqa: BLE001
            continue

    # 2. записи в базе с is_admin=True
    async with get_session() as session:
        res = await session.exec(select(User.tg_id).where(User.is_admin == True))  # noqa: E712
        admin_ids.update(res)

    return list(admin_ids)


@registration_router.message(Command("register"))
async def cmd_register(message: types.Message, state: FSMContext) -> None:
    # если пользователь в списке админ-username — регистрация не требуется
    if message.from_user.username and message.from_user.username.lstrip("@") in [u.lstrip("@") for u in settings.admin_usernames.split()]:
        # убедимся, что запись в БД существует и помечена как админ
        async with get_session() as session:
            result = await session.exec(select(User).where(User.tg_id == message.from_user.id))
            user = result.first()
            if not user:
                user = User(tg_id=message.from_user.id, full_name=message.from_user.full_name or "", is_admin=True, is_approved=True)
                session.add(user)
            else:
                user.is_admin = True
                user.is_approved = True
            await session.commit()

        await message.answer("✅ Вы администратор, регистрация не требуется.")
        return

    async with get_session() as session:
        existing = await session.exec(select(User).where(User.tg_id == message.from_user.id))
        user = existing.first()
        if user and user.is_approved:
            await message.answer("✅ Вы уже зарегистрированы.")
            return

    await message.answer("Введите ваше ФИО:")
    await state.set_state(RegStates.waiting_full_name)


@registration_router.message(RegStates.waiting_full_name, F.text.len() > 3)
async def process_full_name(message: types.Message, state: FSMContext, bot: Bot) -> None:
    full_name = message.text.strip()

    async with get_session() as session:
        stmt = select(User).where(User.tg_id == message.from_user.id)
        result = await session.exec(stmt)
        user = result.first()
        if not user:
            user = User(tg_id=message.from_user.id, full_name=full_name, is_approved=False)
            session.add(user)
            await session.commit()
        else:
            user.full_name = full_name
            await session.commit()

    if user.is_admin:
        # админа подтверждаем автоматически
        async with get_session() as session:
            result = await session.exec(select(User).where(User.tg_id == user.tg_id))
            db_user = result.first()
            if db_user and not db_user.is_approved:
                db_user.is_approved = True
                await session.commit()
        await message.answer("✅ Ваш аккаунт автоматически подтверждён (вы администратор).")
    else:
        await message.answer("Спасибо! Ваши данные отправлены на проверку администратору.")

        # уведомляем админов
        admin_ids = await _get_admin_ids(bot)
        for admin_id in admin_ids:
            await bot.send_message(
                admin_id,
                f"Пользователь {full_name} (id={message.from_user.id}) запрашивает доступ.",
                reply_markup=types.InlineKeyboardMarkup(
                    inline_keyboard=[
                        [
                            types.InlineKeyboardButton(
                                text="✅ Подтвердить", callback_data=f"approve:{message.from_user.id}"
                            ),
                            types.InlineKeyboardButton(
                                text="❌ Отклонить", callback_data=f"reject:{message.from_user.id}"
                            ),
                        ]
                    ]
                ),
            )

    await state.clear()


@registration_router.callback_query(F.data.startswith("approve:"))
async def callback_approve(call: types.CallbackQuery, bot: Bot) -> None:  # noqa: D401
    _, user_id_str = call.data.split(":", 1)
    user_id = int(user_id_str)

    async with get_session() as session:
        result = await session.exec(select(User).where(User.tg_id == user_id))
        user = result.first()
        if not user:
            await call.answer("Пользователь не найден", show_alert=True)
            # Уберём кнопки, чтобы другие админы не пытались снова
            await call.message.edit_reply_markup(reply_markup=None)
            return

        if user.is_approved:
            await call.answer("Уже подтверждён", show_alert=True)
            await call.message.edit_reply_markup(reply_markup=None)
            return

        user.is_approved = True
        await session.commit()

    # Уведомляем пользователя
    try:
        await bot.send_message(user_id, "🎉 Ваш аккаунт подтверждён! Теперь вы можете участвовать в заказах.")
    except Exception:
        pass

    # Убираем кнопки в сообщении, чтобы остальные админы не нажимали
    await call.message.edit_reply_markup(reply_markup=None)
    await call.answer("Пользователь подтверждён")


@registration_router.callback_query(F.data.startswith("reject:"))
async def callback_reject(call: types.CallbackQuery, bot: Bot) -> None:  # noqa: D401
    _, user_id_str = call.data.split(":", 1)
    user_id = int(user_id_str)

    async with get_session() as session:
        result = await session.exec(select(User).where(User.tg_id == user_id))
        user = result.first()

        if not user:
            await call.answer("Заявка уже удалена", show_alert=True)
            await call.message.edit_reply_markup(reply_markup=None)
            return

        if user.is_approved:
            await call.answer("Пользователь уже подтверждён другим админом", show_alert=True)
            await call.message.edit_reply_markup(reply_markup=None)
            return

        await session.delete(user)
        await session.commit()

    try:
        await bot.send_message(user_id, "😔 К сожалению, ваша заявка была отклонена.")
    except Exception:
        pass

    await call.message.edit_reply_markup(reply_markup=None)
    await call.answer("Отклонено") 