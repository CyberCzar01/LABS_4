from __future__ import annotations

import argparse
import os
import subprocess
import sys
import venv
from pathlib import Path
from typing import Dict

VENV_DIR = Path(".venv")
REQ_FILE = Path("requirements.txt")

# ==== Новый код для интерактивного ввода переменных окружения ====
# Пользователь вводит значения через простое GUI-окно (Tkinter).
# Если GUI недоступен (например, скрипт запущен на сервере без дисплея),
# переходим на текстовый ввод в консоли.

ENV_VARS = {
    "BOT_TOKEN": {
        "label": "Токен бота",
        "hint": "Получите у @BotFather",
        "required": True,
    },
    "ADMIN_USERNAMES": {
        "label": "Администраторы (@user …)",
        "hint": "Через пробел, опционально",
        "required": False,
    },
    "DATABASE_URL": {
        "label": "DATABASE_URL",
        "hint": "Оставьте пустым для SQLite",
        "required": False,
    },
    "FEEDBACK_CHAT_ID": {
        "label": "Feedback chat id",
        "hint": "Опционально",
        "required": False,
    },
    "TIMEZONE": {
        "label": "Часовой пояс",
        "hint": "IANA, напр. Europe/Moscow",
        "required": False,
        "default": "Europe/Moscow",
    },
}


def ensure_venv() -> Path:
    if not VENV_DIR.exists():
        print("[setup] Создаю виртуальное окружение .venv …", flush=True)
        venv.create(str(VENV_DIR), with_pip=True)
    else:
        print("[setup] Обнаружено существующее .venv", flush=True)

    python_exe = (
        VENV_DIR / "Scripts" / "python.exe" if os.name == "nt" else VENV_DIR / "bin" / "python"
    )
    if not python_exe.exists():
        sys.exit("Не удалось найти интерпретатор в виртуальном окружении :(")
    return python_exe


def install_requirements(python_exe: Path) -> None:
    if not REQ_FILE.exists():
        print("[warn] Файл requirements.txt не найден — пропускаю установку зависимостей")
        return

    print("[setup] Устанавливаю/обновляю зависимости …", flush=True)
    import subprocess as _sp
    _sp.check_call(
        [str(python_exe), "-m", "pip", "install", "-q", "-r", str(REQ_FILE)],
        stdout=_sp.DEVNULL,
        stderr=_sp.STDOUT,
    )


def load_env(env_path: Path) -> None:
    if not env_path.exists():
        print(f"[warn] Файл {env_path} не найден; использую только переменные окружения shell")
        return

    print(f"[setup] Подгружаю переменные из {env_path}")
    with env_path.open("r", encoding="utf-8") as f:
        for raw in f:
            line = raw.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            key, _, value = line.partition("=")
            key, value = key.strip(), value.strip()
            if key and key not in os.environ:
                os.environ[key] = value


def run_bot(python_exe: Path, extra_env: Dict[str, str] | None = None) -> None:
    print("[run] Запускаю FoodBot …", flush=True)
    env = os.environ.copy()
    if extra_env:
        env.update(extra_env)
    subprocess.call([str(python_exe), "-m", "foodbot.main"], env=env)


def _write_env_file(env_path: Path, data: Dict[str, str]) -> None:
    """Создаёт/перезаписывает .env с парами KEY=VALUE."""
    lines: list[str] = []
    for key, meta in ENV_VARS.items():
        val = data.get(key, "").strip()
        if not val and key == "DATABASE_URL":
            val = "sqlite+aiosqlite:///./foodbot.db"
        if val:
            lines.append(f"{key}={val}")
    env_path.write_text("\n".join(lines), encoding="utf-8")
    print(f"[setup] Файл {env_path} сохранён.")


def _cli_env_setup(env_path: Path) -> None:
    """Fallback: собираем ввод через stdin, если Tkinter не доступен."""
    print("=== Настройка FoodBot ===")
    data: Dict[str, str] = {}
    for key, meta in ENV_VARS.items():
        label = meta["label"]
        default = meta.get("default", "")
        prompt = f"{label} ({default if default else meta.get('hint', '')}): "
        val = input(prompt).strip()
        if not val and default:
            val = default
        if meta.get("required") and not val:
            print("Поле обязательно! Попробуйте ещё раз.")
            return _cli_env_setup(env_path)
        data[key] = val
    _write_env_file(env_path, data)


def interactive_env_setup(env_path: Path) -> None:
    """Открывает GUI-окно для ввода переменных окружения.

    Если не удаётся инициализировать Tk (например, нет DISPLAY),
    переходим на CLI-fallback.
    """
    try:
        import tkinter as tk
        from tkinter import ttk, messagebox
    except Exception:
        return _cli_env_setup(env_path)

    root = tk.Tk()
    root.title("FoodBot — начальная настройка")
    # Центрируем окно
    root.geometry("400x400")

    entries: Dict[str, tk.Entry] = {}

    def _on_submit() -> None:
        data: Dict[str, str] = {}
        for key, ent in entries.items():
            val = ent.get().strip()
            if not val and ENV_VARS[key].get("default"):
                val = str(ENV_VARS[key]["default"])
            if ENV_VARS[key].get("required") and not val:
                messagebox.showerror("Ошибка", f"Поле '{ENV_VARS[key]['label']}' обязательно")
                return
            data[key] = val

        _write_env_file(env_path, data)
        messagebox.showinfo("Готово", "Настройки сохранены. Бот будет запущен.")
        root.destroy()

    frm = ttk.Frame(root, padding=10)
    frm.pack(fill="both", expand=True)

    row = 0
    for key, meta in ENV_VARS.items():
        lbl = ttk.Label(frm, text=meta["label"])
        lbl.grid(column=0, row=row, sticky="w")
        ent = ttk.Entry(frm, width=40)
        if meta.get("default"):
            ent.insert(0, str(meta["default"]))
        ent.grid(column=0, row=row + 1, pady=(0, 10))
        entries[key] = ent
        row += 2

    btn = ttk.Button(frm, text="Сохранить и запустить", command=_on_submit)
    btn.grid(column=0, row=row, pady=10)

    root.mainloop()


def ensure_env(env_path: Path) -> None:
    """Гарантирует наличие корректного .env.

    При отсутствии файла или обязательных переменных запускает мастера.
    """
    load_env(env_path)

    # Если BOT_TOKEN отсутствует или placeholder — просим пользователя ввести данные
    if not os.environ.get("BOT_TOKEN") or os.environ["BOT_TOKEN"].startswith("YOUR_"):
        print("[setup] Требуются параметры для .env …")
        interactive_env_setup(env_path)
        load_env(env_path)  # повторная загрузка после создания


def main() -> None:
    parser = argparse.ArgumentParser(description="FoodBot launcher")
    parser.add_argument(
        "--env",
        "-e",
        default="example.env",
        help="Путь к .env файлу (по умолчанию example.env)",
    )
    args = parser.parse_args()

    python_exe = ensure_venv()
    install_requirements(python_exe)

    env_path = Path(args.env)
    ensure_env(env_path)

    # Проверки переменных окружения
    def need(var: str, hint: str) -> None:
        if not os.environ.get(var) or os.environ[var].startswith("YOUR_"):
            sys.exit(f"Переменная {var} не задана. {hint}")

    need("BOT_TOKEN", "Получите токен у @BotFather и пропишите его в .env или shell.")
    if not os.environ.get("DATABASE_URL"):
        default_sqlite = "sqlite+aiosqlite:///./foodbot.db"
        print(f"[setup] DATABASE_URL не задан → использую {default_sqlite}")
        os.environ["DATABASE_URL"] = default_sqlite

    # Запускаем только Telegram-бота (без Web-API)
    run_bot(python_exe)


if __name__ == "__main__":
    try:
        main()
    except subprocess.CalledProcessError as exc:
        sys.exit(f"Ошибка при выполнении команды: {exc}") 