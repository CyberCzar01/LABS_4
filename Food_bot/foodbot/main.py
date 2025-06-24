import asyncio
import logging

from aiogram import Bot, Dispatcher, Router, types, F
from aiogram.filters import Command
from aiogram.fsm.storage.memory import MemoryStorage
from aiogram.client.default import DefaultBotProperties

from foodbot.config import settings
from foodbot.database import init_db
from foodbot.routers import registration_router, admin_router, menu_router, menu_btn_router, feedback_router


router = Router()


@router.message(Command("start"))
async def cmd_start(message: types.Message) -> None:
    """Приветственное сообщение."""
    # Reply-кнопка «Админ» только для администраторов
    from foodbot.routers.admin import _is_admin  # локальный импорт, чтобы избежать циклов
    if await _is_admin(message.from_user.id, message.from_user.username):
        rkb = types.ReplyKeyboardMarkup(
            keyboard=[[types.KeyboardButton(text="Админ")]], resize_keyboard=True
        )
    else:
        rkb = None

    await message.answer(
        "👋 Привет! Я бот для заказа обедов. Все заказы оформляются только через меню, которое создаёт администратор.\n"
        "Если вы ещё не зарегистрированы — используйте команду /register.",
    )

    if rkb:
        await message.answer("Для быстрого доступа используйте кнопку 'Админ'.", reply_markup=rkb)


@router.message(F.text.lower() == "админ")
async def msg_admin_shortcut(message: types.Message):
    from foodbot.routers.admin import _is_admin, _build_admin_kb

    if not await _is_admin(message.from_user.id, message.from_user.username):
        return
    await message.answer("Панель администратора", reply_markup=_build_admin_kb())


async def _sync_admins(bot: Bot) -> None:
    """Устанавливает флаг is_admin пользователям из ADMIN_USERNAMES."""
    from foodbot.database import get_session  # локальный импорт, чтобы избежать циклов
    from foodbot.models import User
    from sqlmodel import select

    async with get_session() as session:
        for raw_username in settings.admin_usernames.split() or []:
            raw_username = raw_username.strip()
            if not raw_username:
                continue

            try:
                chat = await bot.get_chat(raw_username)  # Telegram API ждёт @username
            except Exception:  # noqa: BLE001
                continue

            username = raw_username.lstrip("@")

            result = await session.exec(select(User).where(User.tg_id == chat.id))
            user = result.first()
            if user:
                user.is_admin = True
                user.is_approved = True  # админа также считаем подтверждённым
            else:
                session.add(User(tg_id=chat.id, full_name=chat.full_name or username, is_admin=True, is_approved=True))
            await session.commit()


async def main() -> None:
    """Точка входа приложения."""
    logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(name)s - %(message)s")

    bot = Bot(token=settings.bot_token, default=DefaultBotProperties(parse_mode="HTML"))
    storage = MemoryStorage()
    dp = Dispatcher(storage=storage)
    dp.include_router(router)  # стартовые команды
    dp.include_router(registration_router)
    dp.include_router(admin_router)
    dp.include_router(menu_router)
    dp.include_router(menu_btn_router)
    dp.include_router(feedback_router)

    logging.info("🚀 Бот запущен…")
    await init_db()

    # На всякий случай снимаем старый webhook, если он был, чтобы Telegram не держал второй getUpdates
    try:
        await bot.delete_webhook(drop_pending_updates=True)
    except Exception as e:  # noqa: BLE001
        logging.warning("Не удалось удалить webhook: %s", e)

    await _sync_admins(bot)

    # запускаем фоновый таск, который закрывает меню по дедлайну
    from foodbot.services.menus import deadline_checker
    asyncio.create_task(deadline_checker(bot))

    await dp.start_polling(bot)


if __name__ == "__main__":
    asyncio.run(main()) 