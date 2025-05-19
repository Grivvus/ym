import pathlib

from pydantic import computed_field
from pydantic_settings import BaseSettings, SettingsConfigDict


class Settings(BaseSettings):
    model_config = SettingsConfigDict(
        env_file=str(pathlib.Path(__file__).parent.parent) + "/.env",
        extra="ignore"
    )
    LOG_LEVEL: str

    APPLICATION_HOST: str
    APPLICATION_PORT: int

    POSTGRES_HOST: str
    POSTGRES_PORT: str
    POSTGRES_PASSWORD: str
    POSTGRES_USER: str
    POSTGRES_DB: str

    JWT_SECRET: str

    S3_HOST: str
    S3_PORT: str
    S3_ROOT_USER: str
    S3_ROOT_PASSWORD: str

    @computed_field
    @property
    def db_url(self) -> str:
        return "postgresql+psycopg://"\
            + f"{self.POSTGRES_USER}:{self.POSTGRES_PASSWORD}@"\
            + f"{self.POSTGRES_HOST}:{self.POSTGRES_PORT}/"\
            + f"{self.POSTGRES_DB}"

    @computed_field
    @property
    def db_url_alembic(self) -> str:
        return "postgresql+psycopg://"\
            + f"{self.POSTGRES_USER}:{self.POSTGRES_PASSWORD}@"\
            + f"0.0.0.0:{self.POSTGRES_PORT}/"\
            + f"{self.POSTGRES_DB}"


settings = Settings()


if __name__ == "__main__":
    print(str(pathlib.Path(__file__).parent.parent))
    print(settings.db_url)
