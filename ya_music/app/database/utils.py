from contextlib import contextmanager
from typing import Generator

from sqlalchemy import Engine, create_engine
from sqlalchemy.orm import Session, sessionmaker

from app.settings import settings

_engine: Engine | None = None


def get_db_egnine(db_url: str | None = None) -> Engine:
    if db_url is None:
        db_url = settings.db_url
    global _engine
    if _engine is None:
        _engine = create_engine(db_url)
    return _engine


@contextmanager
def get_session() -> Generator[Session, None, None]:
    engine = get_db_egnine()
    session_factory = sessionmaker(engine, autoflush=False, autocommit=False)

    session = session_factory()
    try:
        yield session
    finally:
        session.close()
