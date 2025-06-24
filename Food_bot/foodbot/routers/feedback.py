from __future__ import annotations

from aiogram import Router, types, F
from aiogram.filters import Command
from aiogram.fsm.context import FSMContext
from aiogram.fsm.state import State, StatesGroup

from foodbot.config import settings

feedback_router = Router()


class FbStates(StatesGroup):
    waiting_text = State()


@feedback_router.message(Command("feedback"))
async def cmd_feedback(message: types.Message, state: FSMContext):
    await message.answer("Напишите сообщение, и я передам его администратору. Чтобы отменить, отправьте /cancel")
    await state.set_state(FbStates.waiting_text)


@feedback_router.message(FbStates.waiting_text, F.text.len() > 0)
async def fb_receive(message: types.Message, state: FSMContext):
    text = message.text.strip()
    fb_chat = settings.feedback_chat_id
    if not fb_chat:
        await message.answer("Извините, чат для обратной связи не настроен.")
    else:
        from datetime import datetime
        ts = datetime.now().strftime('%d.%m %H:%M')
        header = f"<b>Feedback</b> от <a href=\"tg://user?id={message.from_user.id}\">{message.from_user.full_name}</a> ({message.from_user.id})\n<code>{ts}</code>"
        await message.bot.send_message(fb_chat, f"{header}\n\n{text}", parse_mode="HTML")
        await message.answer("✅ Спасибо! Сообщение передано.")
    await state.clear() 