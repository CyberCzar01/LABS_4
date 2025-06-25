"""Роутер команд администратора."""

from aiogram import Router, types, F
from aiogram.filters import Command
from aiogram import Bot
from sqlmodel import select
from sqlalchemy import delete as sa_delete

from foodbot.models import User, Canteen, Meal, Order, DailyMenu, MenuItem
from foodbot.database import get_session
from foodbot.config import settings

from aiogram.fsm.state import State, StatesGroup
from aiogram.fsm.context import FSMContext

from datetime import datetime, timedelta, timezone
from zoneinfo import ZoneInfo

from foodbot.services.orders import build_report

admin_router = Router()


async def _is_admin(tg_id: int, username: str | None = None) -> bool:
    """Проверяет и при необходимости помечает пользователя как админа.

     Если пользователь найден и уже админ — True.
     Если пользователя нет, но его username присутствует в ADMIN_USERNAMES,
     создаём запись с флагами is_admin/is_approved и возвращаем True.
     Во всех остальных случаях — False.
    """

    admin_usernames = [u.lstrip("@").lower() for u in settings.admin_usernames.split()]

    async with get_session() as session:
        result = await session.exec(select(User).where(User.tg_id == tg_id))
        user = result.first()

        if user and user.is_admin:
            return True

        # Возможен вариант: запись есть, но флаги не выставлены
        if username:
            norm_username = username.lstrip("@").lower()
            if norm_username in admin_usernames:
                if not user:
                    user = User(
                        tg_id=tg_id,
                        full_name=username,
                        is_admin=True,
                        is_approved=True,
                    )
                    session.add(user)
                else:
                    user.is_admin = True
                    user.is_approved = True
                await session.commit()
                return True

    return False


@admin_router.message(Command("admin"))
async def cmd_admin(message: types.Message, bot: Bot) -> None:
    if not await _is_admin(message.from_user.id, message.from_user.username):
        await message.reply("У вас нет прав администратора.")
        return

    # Inline-меню
    await message.answer("Панель администратора", reply_markup=_build_admin_kb())

    # Ставим reply-кнопку «Админ» без лишнего текста, чтобы не засорять чат
    admin_rkb = types.ReplyKeyboardMarkup(keyboard=[[types.KeyboardButton(text="Админ")]], resize_keyboard=True)
    await message.answer("\u2060", reply_markup=admin_rkb)


@admin_router.callback_query(lambda c: c.data == "list_users")
async def callback_list_users(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    async with get_session() as session:
        users = await session.exec(select(User).where(User.is_approved == True))  # noqa: E712
        text = "Подтверждённые пользователи:\n\n" + "\n".join(
            f"{u.full_name} (id {u.tg_id})" for u in users
        )

    kb = types.InlineKeyboardMarkup(
        inline_keyboard=[
            [types.InlineKeyboardButton(text="⬅️ Назад", callback_data="admin")],
            _home_row(),
        ]
    )
    await call.message.answer(text or "Пока нет пользователей", reply_markup=kb)
    await call.answer()


#  Управление столовыми


class CanteenAddStates(StatesGroup):
    waiting_title = State()


@admin_router.callback_query(lambda c: c.data == "list_canteens")
async def callback_list_canteens(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    async with get_session() as session:
        result = await session.exec(select(Canteen))
        canteens = list(result)
        # Кнопки: заголовок
        kb_rows: list[list[types.InlineKeyboardButton]] = []
        for c in canteens:
            status_btn = types.InlineKeyboardButton(
                text="🟢" if c.is_active else "🔴",
                callback_data=f"canteen_toggle:{c.id}",
            )
            del_btn = types.InlineKeyboardButton(text="🗑", callback_data=f"canteen_del:{c.id}")
            title_btn = types.InlineKeyboardButton(text=c.title, callback_data=f"canteen:{c.id}")
            kb_rows.append([title_btn, status_btn, del_btn])

        # Текстовое представление списка столовых для отправки сообщением
        rows = [f"{c.id}. {c.title} {'(🔴)' if not c.is_active else ''}" for c in canteens]

    # Кнопка «Добавить столовую» перед навигацией
    kb_rows.append([types.InlineKeyboardButton(text="➕ Добавить столовую", callback_data="canteen_add")])

    # Добавляем навигацию
    kb_rows.append([types.InlineKeyboardButton(text="⬅️ Назад", callback_data="admin")])
    kb_rows.append(_home_row())

    text = "Столовые:\n" + ("\n".join(rows) if rows else "Нет ни одной столовой")
    kb = types.InlineKeyboardMarkup(inline_keyboard=kb_rows)

    try:
        # Пытаемся отредактировать существующее сообщение
        await call.message.edit_text(text)
        await call.message.edit_reply_markup(reply_markup=kb)
    except Exception:  # если не удалось, то шлём новое
        await call.message.answer(text, reply_markup=kb)
    await call.answer()


@admin_router.callback_query(lambda c: c.data == "canteen_add")
async def callback_canteen_add(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    await state.set_state(CanteenAddStates.waiting_title)
    await call.message.answer("Введите название новой столовой:")
    await call.answer()


@admin_router.message(CanteenAddStates.waiting_title)
async def canteen_add_title(message: types.Message, state: FSMContext):
    if not await _is_admin(message.from_user.id, message.from_user.username):
        await message.reply("Нет прав администратора.")
        return

    title = message.text.strip()
    if not title:
        await message.reply("Название не может быть пустым. Попробуйте ещё раз.")
        return

    async with get_session() as session:
        session.add(Canteen(title=title))
        await session.commit()

    await message.answer(f"✅ Столовая «{title}» добавлена.")

    # Показываем обновлённый список столовых, чтобы админ мог продолжить работу
    from types import SimpleNamespace

    async def _dummy_answer(*args, **kwargs):
        return None

    fake_call = SimpleNamespace(message=message, from_user=message.from_user, data="list_canteens", answer=_dummy_answer)
    await callback_list_canteens(fake_call, state)  # type: ignore[arg-type]

    await state.clear()


#  Блюда


class MealAddStates(StatesGroup):
    waiting_title = State()
    waiting_desc = State()
    waiting_complex = State()


class MenuCreateStates(StatesGroup):
    """Состояния мастера создания меню дня."""

    waiting_canteen = State()  # выбор столовых
    waiting_deadline = State()  # ввод времени дедлайна


# Показать список блюд столовой


@admin_router.callback_query(lambda c: c.data and c.data.startswith("canteen:"))
async def callback_canteen_meals(call: types.CallbackQuery, state: FSMContext):
    """Показывает блюда выбранной столовой с пагинацией.

    callback_data может быть двух форматов:
    "canteen:<id>" – страница 0 (по умолчанию)
    "canteen:<id>:<page>" – нужная страница (0-based)
    """

    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    parts = call.data.split(":")
    if len(parts) == 2:
        _, canteen_id_str = parts
        page = 0
    else:
        _, canteen_id_str, page_str = parts
        page = int(page_str)

    canteen_id = int(canteen_id_str)

    PAGE_SIZE = 10

    async with get_session() as session:
        canteen = await session.get(Canteen, canteen_id)
        if not canteen:
            await call.answer("Столовая не найдена", show_alert=True)
            return

        meals = list(await session.exec(select(Meal).where(Meal.canteen_id == canteen_id)))

    total_pages = (len(meals) - 1) // PAGE_SIZE + 1 if meals else 1
    page = max(0, min(page, total_pages - 1))  # clamp
    slice_start = page * PAGE_SIZE
    slice_end = slice_start + PAGE_SIZE
    page_meals = meals[slice_start:slice_end]

    rows_txt = [
        f"{m.id}. {m.title} {'(🔴)' if not m.is_active else ''}" for m in page_meals
    ]
    text = (
        f"Меню столовой «{canteen.title}» (стр. {page + 1}/{total_pages}):\n"
        + ("\n".join(rows_txt) if rows_txt else "Пока пусто")
    )

    kb_rows: list[list[types.InlineKeyboardButton]] = []
    for m in page_meals:
        # Одна строка: название | статус | удалить  – чтобы не утроять кол-во кнопок
        info_btn = types.InlineKeyboardButton(text=m.title[:30], callback_data=f"meal_info:{m.id}:{canteen_id}")
        status_btn = types.InlineKeyboardButton(
            text="🟢" if m.is_active else "🔴",
            callback_data=f"meal_toggle:{m.id}:{canteen_id}",
        )
        del_btn = types.InlineKeyboardButton(text="🗑", callback_data=f"meal_del:{m.id}:{canteen_id}")
        kb_rows.append([info_btn, status_btn, del_btn])

    # навигация страниц
    nav_row: list[types.InlineKeyboardButton] = []
    if page > 0:
        nav_row.append(
            types.InlineKeyboardButton(text="◀️", callback_data=f"canteen:{canteen_id}:{page - 1}")
        )
    if page < total_pages - 1:
        nav_row.append(
            types.InlineKeyboardButton(text="▶️", callback_data=f"canteen:{canteen_id}:{page + 1}")
        )
    if nav_row:
        kb_rows.append(nav_row)

    # прочие действия
    kb_rows.append([
        types.InlineKeyboardButton(text="➕ Добавить блюдо", callback_data=f"meal_add:{canteen_id}"),
    ])
    kb_rows.append([types.InlineKeyboardButton(text="⬅️ Назад", callback_data="list_canteens")])
    kb_rows.append(_home_row())

    kb = types.InlineKeyboardMarkup(inline_keyboard=kb_rows)

    try:
        await call.message.edit_text(text, reply_markup=kb)
    except Exception:
        await call.message.answer(text, reply_markup=kb)
    await call.answer()


# Запрос добавления блюда


@admin_router.callback_query(lambda c: c.data and c.data.startswith("meal_add:"))
async def callback_meal_add(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, canteen_id_str = call.data.split(":", 1)
    await state.update_data(canteen_id=int(canteen_id_str))
    await state.set_state(MealAddStates.waiting_title)
    await call.message.answer("Введите название блюда:")
    await call.answer()


# Приём названия блюда


@admin_router.message(MealAddStates.waiting_title)
async def meal_add_title(message: types.Message, state: FSMContext):
    if not await _is_admin(message.from_user.id, message.from_user.username):
        await message.reply("Нет прав администратора.")
        return

    data = await state.get_data()
    canteen_id: int | None = data.get("canteen_id")
    title = message.text.strip()

    if not title:
        await message.reply("Название не может быть пустым. Попробуйте ещё раз.")
        return

    await state.update_data(new_meal_title=title, canteen_id=canteen_id)
    await state.set_state(MealAddStates.waiting_desc)
    await message.answer("Введите описание блюда (или - если пусто):")


@admin_router.message(MealAddStates.waiting_desc)
async def meal_add_desc(message: types.Message, state: FSMContext):
    desc = "" if message.text.strip() == "-" else message.text.strip()
    await state.update_data(new_meal_desc=desc)

    # Просим выбрать тип блюда — комплекс или обычное
    kb = types.ReplyKeyboardMarkup(
        keyboard=[[types.KeyboardButton(text="Комплексный обед"), types.KeyboardButton(text="Обычное блюдо")]],
        resize_keyboard=True,
        one_time_keyboard=True,
    )
    await message.answer("Выберите тип блюда:", reply_markup=kb)
    await state.set_state(MealAddStates.waiting_complex)


@admin_router.message(MealAddStates.waiting_complex)
async def meal_add_complex(message: types.Message, state: FSMContext):
    txt = message.text.strip().lower()
    if txt not in {"комплексный обед", "обычное блюдо"}:
        await message.reply("Пожалуйста, выберите из предложенных вариантов: 'Комплексный обед' или 'Обычное блюдо'.")
        return

    is_complex = txt.startswith("комплекс")
    data = await state.get_data()
    title = data.get("new_meal_title")
    desc = data.get("new_meal_desc", "")
    canteen_id = data.get("canteen_id")

    async with get_session() as session:
        session.add(Meal(title=title, description=desc, is_complex=is_complex, canteen_id=canteen_id))
        await session.commit()

    # Убираем временную клавиатуру выбора типа блюда
    await message.answer(f"✅ Блюдо «{title}» добавлено.", reply_markup=types.ReplyKeyboardRemove())

    # Возвращаем кнопки: inline-панель и reply-кнопку «Админ» для быстрого доступа
    await message.answer("Панель администратора", reply_markup=_build_admin_kb())

    admin_rkb = types.ReplyKeyboardMarkup(
        keyboard=[[types.KeyboardButton(text="Админ")]], resize_keyboard=True
    )
    await message.answer("\u2060", reply_markup=admin_rkb)

    await state.clear()


#  Отчёты


@admin_router.callback_query(lambda c: c.data == "report_today")
async def callback_report_today(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    try:
        filename, buffer = await build_report(datetime.now())
    except ValueError:
        await call.message.answer("Сегодня заказов ещё нет.")
        await call.answer()
        return

    file = types.BufferedInputFile(buffer.read(), filename=filename)
    await call.message.answer_document(file, caption="Отчёт за сегодня")
    # Возвращаем меню админа
    await call.message.answer("Панель администратора", reply_markup=_build_admin_kb())
    await call.answer()


# Управление пользователями


@admin_router.callback_query(lambda c: c.data == "manage_users")
async def callback_manage_users(call: types.CallbackQuery):
    """Показывает список пользователей с возможностью управления."""
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    # Сразу подтверждаем callback, чтобы Telegram не выдал ошибку timeout > 3 сек
    await call.answer()

    async with get_session() as session:
        users_result = await session.exec(select(User))
        users = list(users_result)

    if not users:
        await call.message.answer("Пользователей пока нет")
        return

    kb_rows: list[list[types.InlineKeyboardButton]] = []
    for u in users:
        label = f"{u.full_name or u.tg_id} {'⭐' if u.is_admin else ''}"
        kb_rows.append([
            types.InlineKeyboardButton(text=label, callback_data=f"user_mng:{u.id}"),
        ])

    kb_rows.append([types.InlineKeyboardButton(text="⬅️ Назад", callback_data="admin")])
    kb_rows.append(_home_row())
    kb = types.InlineKeyboardMarkup(inline_keyboard=kb_rows)
    await call.message.answer("Выберите пользователя:", reply_markup=kb)


@admin_router.callback_query(lambda c: c.data and c.data.startswith("user_mng:"))
async def callback_user_manage(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    # Подтверждаем callback сразу
    await call.answer()

    _, user_id_str = call.data.split(":", 1)
    user_id = int(user_id_str)

    async with get_session() as session:
        user = await session.get(User, user_id)
        if not user:
            await call.message.answer("Пользователь не найден")
            return

        text = f"<b>{user.full_name}</b>\nTG ID: <code>{user.tg_id}</code>\nАдмин: {'да' if user.is_admin else 'нет'}"

    kb = types.InlineKeyboardMarkup(
        inline_keyboard=[
            [types.InlineKeyboardButton(text="❌ Удалить", callback_data=f"user_del:{user_id}")],
            [
                types.InlineKeyboardButton(
                    text="⬆️ Сделать админом" if not user.is_admin else "⬇️ Снять админа",
                    callback_data=f"user_toggle_admin:{user_id}",
                )
            ],
            [types.InlineKeyboardButton(text="⬅️ Назад", callback_data="manage_users")],
            _home_row(),
        ]
    )
    await call.message.answer(text, reply_markup=kb, parse_mode="HTML")


@admin_router.callback_query(lambda c: c.data and c.data.startswith("user_del:"))
async def callback_user_delete(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, user_id_str = call.data.split(":", 1)
    user_id = int(user_id_str)

    async with get_session() as session:
        user = await session.get(User, user_id)
        if not user:
            await call.answer("Пользователь не найден", show_alert=True)
            return

        # сначала удалим связанные заказы, затем пользователя
        await session.exec(sa_delete(Order).where(Order.user_id == user.id))
        await session.delete(user)
        await session.commit()

    await call.message.answer("✅ Пользователь деактивирован")
    await call.answer()


@admin_router.callback_query(lambda c: c.data and c.data.startswith("user_toggle_admin:"))
async def callback_user_toggle_admin(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, user_id_str = call.data.split(":", 1)
    user_id = int(user_id_str)

    async with get_session() as session:
        user = await session.get(User, user_id)
        if not user:
            await call.answer("Пользователь не найден", show_alert=True)
            return

        user.is_admin = not user.is_admin
        await session.commit()

    await call.message.answer(
        "✅ Пользователь теперь админ" if user.is_admin else "✅ Права админа сняты"
    )
    await call.answer()


#  Опросы


@admin_router.callback_query(lambda c: c.data == "start_poll")
async def callback_start_poll(call: types.CallbackQuery, bot: Bot):
    """Создаёт быстрый опрос по активным блюдам (первые 10)."""
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    async with get_session() as session:
        result = await session.exec(
            select(Meal).where(Meal.is_active == True).limit(10)  # noqa: E712
        )
        meals = list(result)

    if not meals:
        await call.answer("Нет активных блюд", show_alert=True)
        return

    options = [m.title for m in meals]

    poll_msg = await bot.send_poll(
        chat_id=call.message.chat.id,
        question="Что будем заказывать сегодня?",
        options=options,
        is_anonymous=False,
    )

    # Предлагаем вернуться в панель админа
    await call.message.answer("✅ Опрос создан!", reply_markup=_build_admin_kb())
    await call.answer()


# Меню дня


# Шаг 1. выбор столовых


@admin_router.callback_query(lambda c: c.data == "start_menu")
async def callback_start_menu(call: types.CallbackQuery, bot: Bot, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    # Загружаем активные столовые
    async with get_session() as session:
        res = await session.exec(select(Canteen).where(Canteen.is_active == True))
        canteens = list(res)

    if not canteens:
        await call.answer("Нет активных столовых", show_alert=True)
        return

    # инициализируем список выбранных столовых
    await state.set_state(MenuCreateStates.waiting_canteen)
    await state.update_data(sel_cids=[])

    async def _send_picker(selected: list[int]):
        kb_rows: list[list[types.InlineKeyboardButton]] = []
        for c in canteens:
            marker = "🟢" if c.id in selected else "⚪️"
            kb_rows.append([
                types.InlineKeyboardButton(text=f"{marker} {c.title}", callback_data=f"menu_cant_toggle:{c.id}"),
            ])

        # кнопка "Все столовые" — зелёная если выбраны все
        all_selected = len(selected) == len(canteens)
        all_marker = "🟢" if all_selected else "⚪️"
        kb_rows.append([
            types.InlineKeyboardButton(text=f"{all_marker} Все столовые", callback_data="menu_cant_all"),
        ])

        # Кнопка «Отменить выбор»
        if selected:
            kb_rows.append([
                types.InlineKeyboardButton(text="❌ Отменить выбор", callback_data="menu_cant_reset"),
            ])

        # Кнопка «Далее» только если есть выбор
        if selected:
            kb_rows.append([
                types.InlineKeyboardButton(text="✅ Далее", callback_data="menu_cant_next"),
            ])

        await call.message.answer(
            "Отметьте нужные столовые (можно несколько):",
            reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows),
        )

    await _send_picker([])
    await call.answer()


# Шаг 2. выбрана столовая


@admin_router.callback_query(MenuCreateStates.waiting_canteen, lambda c: c.data and c.data.startswith("menu_cant_toggle:"))
async def callback_menu_cant_toggle(call: types.CallbackQuery, state: FSMContext):
    _, cid_str = call.data.split(":", 1)
    cid = int(cid_str)

    data = await state.get_data()
    selected: list[int] = data.get("sel_cids", [])
    if cid in selected:
        selected.remove(cid)
    else:
        selected.append(cid)
    await state.update_data(sel_cids=selected)

    # перерисовываем список (используем ту же логику, что и в start_menu)
    # Чтобы не дублировать код, отправим новый call с data="start_menu_redraw"
    new_call = call.model_copy(update={"data": "start_menu_redraw"})
    await callback_start_menu_redraw(new_call, state)  # type: ignore[arg-type]


@admin_router.callback_query(MenuCreateStates.waiting_canteen, lambda c: c.data == "start_menu_redraw")
async def callback_start_menu_redraw(call: types.CallbackQuery, state: FSMContext):
    """Вспомогательный хендлер: перерисовывает список столовых c чекбоксами."""
    # Загружаем необходимые данные
    data = await state.get_data()
    selected: list[int] = data.get("sel_cids", [])

    async with get_session() as session:
        res = await session.exec(select(Canteen).where(Canteen.is_active == True))
        canteens = list(res)

    kb_rows: list[list[types.InlineKeyboardButton]] = []
    for c in canteens:
        marker = "🟢" if c.id in selected else "⚪️"
        kb_rows.append([
            types.InlineKeyboardButton(text=f"{marker} {c.title}", callback_data=f"menu_cant_toggle:{c.id}"),
        ])
    all_selected = len(selected) == len(canteens)
    all_marker = "🟢" if all_selected else "⚪️"
    kb_rows.append([
        types.InlineKeyboardButton(text=f"{all_marker} Все столовые", callback_data="menu_cant_all"),
    ])
    if selected:
        kb_rows.append([
            types.InlineKeyboardButton(text="❌ Отменить выбор", callback_data="menu_cant_reset"),
        ])
    if selected:
        kb_rows.append([
            types.InlineKeyboardButton(text="✅ Далее", callback_data="menu_cant_next"),
        ])

    try:
        await call.message.edit_reply_markup(
            reply_markup=types.InlineKeyboardMarkup(inline_keyboard=kb_rows)
        )
    except Exception:
        pass
    await call.answer()


# Далее – переходим к дедлайну


@admin_router.callback_query(MenuCreateStates.waiting_canteen, lambda c: c.data == "menu_cant_next")
async def callback_menu_cant_next(call: types.CallbackQuery, state: FSMContext):
    data = await state.get_data()
    selected: list[int] = data.get("sel_cids", [])
    if not selected:
        await call.answer("Нужно выбрать хотя бы одну столовую", show_alert=True)
        return

    await state.update_data(canteen_sel=selected)  # сохраняем список
    await state.set_state(MenuCreateStates.waiting_deadline)
    await call.message.answer("Введите время окончания опроса (HH:MM, напр. 16:00):")
    await call.answer()


@admin_router.message(MenuCreateStates.waiting_deadline)
async def menu_set_deadline(message: types.Message, state: FSMContext, bot: Bot):
    if not await _is_admin(message.from_user.id, message.from_user.username):
        await message.reply("Нет прав администратора.")
        await state.clear()
        return

    text = message.text.strip()
    try:
        hour, minute = map(int, text.split(":", 1))
        if not (0 <= hour <= 23 and 0 <= minute <= 59):
            raise ValueError
    except ValueError:
        await message.reply("Неверный формат времени. Используйте HH:MM, напр. 16:00")
        return

    tz = settings.tz

    now_local = datetime.now(tz)
    deadline_local = now_local.replace(hour=hour, minute=minute, second=0, microsecond=0)
    if deadline_local <= now_local:
        deadline_local += timedelta(days=1)

    deadline = deadline_local.astimezone(timezone.utc)  # храним в UTC

    data = await state.get_data()
    canteen_sel = data.get("canteen_sel")
    if not canteen_sel:
        await message.reply("Ошибка: не удалось получить выбор столовой.")
        await state.clear()
        return

    # формируем список актуальных блюд
    async with get_session() as session:
        if canteen_sel == "all":
            res = await session.exec(select(Meal).where(Meal.is_active == True))  # noqa: E712
        elif isinstance(canteen_sel, list):
            ids = [int(x) for x in canteen_sel]
            res = await session.exec(select(Meal).where(Meal.canteen_id.in_(ids), Meal.is_active == True))  # noqa: E712
        else:
            cid = int(canteen_sel)
            res = await session.exec(select(Meal).where(Meal.canteen_id == cid, Meal.is_active == True))  # noqa: E712
        meals = list(res)

    if not meals:
        await message.reply("В выбранной столовой нет активных блюд")
        await state.clear()
        return

    meal_ids = [m.id for m in meals]

    from foodbot.services.menus import create_menu

    # 1. Публикуем меню в текущий чат (может быть группой админов)
    primary_menu = await create_menu(bot=bot, chat_id=message.chat.id, deadline=deadline, meal_ids=meal_ids, primary=True)

    # 2. Рассылаем копии всем подтверждённым пользователям
    from foodbot.models import User
    async with get_session() as session:
        users_res = await session.exec(select(User).where(User.is_approved == True))  # noqa: E712
        users = list(users_res)

    # Отправляем индивидуально; игнорируем ошибки (например, пользователь заблокировал бота)
    for u in users:
        if u.tg_id == message.chat.id:
            continue  # уже отправили выше
        try:
            await create_menu(bot=bot, chat_id=u.tg_id, deadline=deadline, meal_ids=meal_ids, parent_id=primary_menu.id)
        except Exception as e:  # noqa: BLE001
            import logging
            logging.warning("Не удалось отправить меню пользователю %s: %s", u.tg_id, e)

    await message.reply(
        "✅ Меню опубликовано и разослано пользователям. Приём заказов завершится в " + deadline_local.strftime("%H:%M")
    )
    await state.clear()


# Возврат в главное меню админа


def _build_admin_kb() -> types.InlineKeyboardMarkup:
    rows = [
        [types.InlineKeyboardButton(text="👥 Пользователи", callback_data="manage_users")],
        [types.InlineKeyboardButton(text="🏢 Столовые", callback_data="list_canteens")],
        [types.InlineKeyboardButton(text="🗳️ Опрос", callback_data="start_menu")],
        [types.InlineKeyboardButton(text="🏁 Завершить опрос", callback_data="menu_close")],
        [types.InlineKeyboardButton(text="📊 Отчёт (сегодня)", callback_data="report_today")],
    ]
    return types.InlineKeyboardMarkup(inline_keyboard=rows)


@admin_router.callback_query(lambda c: c.data == "admin")
async def callback_admin_menu(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    await call.message.answer("Панель администратора", reply_markup=_build_admin_kb())

    admin_rkb = types.ReplyKeyboardMarkup(keyboard=[[types.KeyboardButton(text="Админ")]], resize_keyboard=True)
    await call.message.answer("\u2060", reply_markup=admin_rkb)

    await call.answer()


# универсальная «Меню»


def _home_row() -> list[types.InlineKeyboardButton]:
    return [types.InlineKeyboardButton(text="🏠 Главное меню", callback_data="admin")]


#  Досрочное закрытие меню


@admin_router.callback_query(lambda c: c.data == "menu_close")
async def callback_menu_close(call: types.CallbackQuery, bot: Bot):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    from datetime import datetime, timezone
    from foodbot.services.menus import close_menu

    async with get_session() as session:
        now = datetime.utcnow().replace(tzinfo=timezone.utc)
        res = await session.exec(
            select(DailyMenu).where(
                DailyMenu.chat_id == call.message.chat.id,
                DailyMenu.is_closed == False,  # noqa: E712
                DailyMenu.deadline > now,
            )
        )
        menu = res.first()

    if not menu:
        await call.answer("Нет активного меню", show_alert=True)
        return

    await close_menu(bot, menu)
    await call.answer("Меню закрыто")


#  Копирование предыдущего меню


@admin_router.callback_query(lambda c: c.data == "menu_copy")
async def callback_menu_copy(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    from datetime import datetime, timezone
    async with get_session() as session:
        res = await session.exec(
            select(DailyMenu).where(DailyMenu.chat_id == call.message.chat.id).order_by(DailyMenu.id.desc())
        )
        last_menu = res.first()

    if not last_menu:
        await call.answer("Предыдущее меню не найдено", show_alert=True)
        return

    async with get_session() as session:
        items_res = await session.exec(select(MenuItem).where(MenuItem.menu_id == last_menu.id))
        meal_ids = [itm.meal_id for itm in items_res]

    if not meal_ids:
        await call.answer("Предыдущее меню пусто", show_alert=True)
        return

    await state.update_data(meal_ids=meal_ids)
    await state.set_state(MenuCreateStates.waiting_deadline)
    await call.message.answer("Введите время окончания опроса (HH:MM, напр. 16:00):")
    await call.answer()


#  Тоггл/удаление столовых


@admin_router.callback_query(lambda c: c.data and c.data.startswith("canteen_toggle:"))
async def callback_canteen_toggle(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, cid_str = call.data.split(":", 1)
    cid = int(cid_str)

    async with get_session() as session:
        canteen = await session.get(Canteen, cid)
        if not canteen:
            await call.answer("Столовая не найдена", show_alert=True)
            return
        canteen.is_active = not canteen.is_active
        await session.commit()

    await call.answer("Статус обновлён")
    await callback_list_canteens(call, state)


@admin_router.callback_query(lambda c: c.data and c.data.startswith("canteen_del:"))
async def callback_canteen_delete(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, cid_str = call.data.split(":", 1)
    cid = int(cid_str)

    async with get_session() as session:
        canteen = await session.get(Canteen, cid)
        if not canteen:
            await call.answer("Не найдено", show_alert=True)
            return
        await session.delete(canteen)
        await session.commit()

    await call.answer("Удалено")
    await callback_list_canteens(call, state)


#  Тоггл блюда


@admin_router.callback_query(lambda c: c.data and c.data.startswith("meal_toggle:"))
async def callback_meal_toggle(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, rest = call.data.split(":", 1)
    meal_id_str, canteen_id_str = rest.split(":", 1)
    meal_id = int(meal_id_str)
    canteen_id = int(canteen_id_str)

    async with get_session() as session:
        meal = await session.get(Meal, meal_id)
        if not meal:
            await call.answer("Не найдено", show_alert=True)
            return
        meal.is_active = not meal.is_active
        await session.commit()

    await call.answer("Статус блюда обновлён")

    # Создаём копию callback с data="canteen:<id>" для корректного обновления списка
    new_call = call.model_copy(update={"data": f"canteen:{canteen_id}"})
    await callback_canteen_meals(new_call, state)


# ===== Удаление блюда =====


@admin_router.callback_query(lambda c: c.data and c.data.startswith("meal_del:"))
async def callback_meal_delete(call: types.CallbackQuery, state: FSMContext):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, rest = call.data.split(":", 1)
    meal_id_str, canteen_id_str = rest.split(":", 1)
    meal_id = int(meal_id_str)
    canteen_id = int(canteen_id_str)

    async with get_session() as session:
        meal = await session.get(Meal, meal_id)
        if not meal:
            await call.answer("Не найдено", show_alert=True)
            return
        # Удаляем связанные заказы
        await session.exec(sa_delete(Order).where(Order.meal_id == meal.id))
        await session.delete(meal)
        await session.commit()

    await call.answer("Блюдо удалено")

    # Обновляем список блюд столовой корректной копией
    new_call = call.model_copy(update={"data": f"canteen:{canteen_id}"})
    await callback_canteen_meals(new_call, state)


# ===== Информация о блюде =====


@admin_router.callback_query(lambda c: c.data and c.data.startswith("meal_info:"))
async def callback_meal_info(call: types.CallbackQuery):
    if not await _is_admin(call.from_user.id, call.from_user.username):
        await call.answer("Нет прав", show_alert=True)
        return

    _, rest = call.data.split(":", 1)
    meal_id_str, canteen_id_str = rest.split(":", 1)
    meal_id = int(meal_id_str)
    canteen_id = int(canteen_id_str)

    async with get_session() as session:
        meal = await session.get(Meal, meal_id)
        if not meal:
            await call.answer("Не найдено", show_alert=True)
            return

    text = f"<b>{meal.title}</b>\n\n{meal.description or 'Описание отсутствует'}\n\nТип: {'комплексный обед' if meal.is_complex else 'блюдо'}"

    kb = types.InlineKeyboardMarkup(
        inline_keyboard=[
            [types.InlineKeyboardButton(text="⬅️ Назад", callback_data=f"canteen:{canteen_id}")],
            _home_row(),
        ]
    )
    await call.message.answer(text, parse_mode="HTML", reply_markup=kb)
    await call.answer()


# === «Все столовые» — быстрое завершение выбора ===


@admin_router.callback_query(MenuCreateStates.waiting_canteen, lambda c: c.data == "menu_cant_all")
async def callback_menu_cant_all(call: types.CallbackQuery, state: FSMContext):
    """Выбрать все столовые (просто отмечает, но не переходит далее)."""
    # Получаем список всех активных столовых
    async with get_session() as session:
        res = await session.exec(select(Canteen.id).where(Canteen.is_active == True))
        all_ids = list(res)

    await state.update_data(sel_cids=all_ids)

    # Триггерим перерисовку
    new_call = call.model_copy(update={"data": "start_menu_redraw"})
    await callback_start_menu_redraw(new_call, state)  # type: ignore[arg-type]


# === Сброс выбора ===


@admin_router.callback_query(MenuCreateStates.waiting_canteen, lambda c: c.data == "menu_cant_reset")
async def callback_menu_cant_reset(call: types.CallbackQuery, state: FSMContext):
    """Отменяет выбор всех столовых."""
    await state.update_data(sel_cids=[])
    # Перерисовать
    new_call = call.model_copy(update={"data": "start_menu_redraw"})
    await callback_start_menu_redraw(new_call, state)  # type: ignore[arg-type] 