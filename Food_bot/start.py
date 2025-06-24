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
    load_env(Path(args.env))

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