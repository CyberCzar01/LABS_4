from pydantic_settings import BaseSettings
from pydantic import Field
from zoneinfo import ZoneInfo, ZoneInfoNotFoundError


class Settings(BaseSettings):
    """Конфигурация проекта, загружаемая из переменных среды."""

    bot_token: str = Field(..., env="BOT_TOKEN")
    admin_usernames: str = Field("", env="ADMIN_USERNAMES")
    database_url: str = Field("sqlite+aiosqlite:///./foodbot.db", env="DATABASE_URL")
    feedback_chat_id: int | None = Field(None, env="FEEDBACK_CHAT_ID")
    timezone: str = Field("Europe/Moscow", env="TIMEZONE")  # IANA TZ

    @property
    def tz(self):
        """Возвращает объект ZoneInfo для текущего часового пояса.

        На Windows библиотеки zoneinfo нет в системе, поэтому
        требуется пакет tzdata (добавлен в requirements). Если
        даже после этого ключ не найден — используем UTC.
        """
        try:
            return ZoneInfo(self.timezone)
        except ZoneInfoNotFoundError:
            import logging
            logging.warning("[config] TIMEZONE '%s' не найден – использую UTC", self.timezone)
            return ZoneInfo("UTC")

    class Config:
        env_file = ".env"
        env_file_encoding = "utf-8"


settings = Settings() 