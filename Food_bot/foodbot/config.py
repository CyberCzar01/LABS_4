from pydantic_settings import BaseSettings
from pydantic import Field


class Settings(BaseSettings):
    """Конфигурация проекта, загружаемая из переменных среды."""

    bot_token: str = Field(..., env="BOT_TOKEN")
    admin_usernames: str = Field("", env="ADMIN_USERNAMES")
    database_url: str = Field("sqlite+aiosqlite:///./foodbot.db", env="DATABASE_URL")
    feedback_chat_id: int | None = Field(None, env="FEEDBACK_CHAT_ID")
    timezone: str = Field("Europe/Moscow", env="TIMEZONE")  # IANA TZ

    class Config:
        env_file = "example.env"
        env_file_encoding = "utf-8"


settings = Settings() 