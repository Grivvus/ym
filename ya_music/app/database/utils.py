import logging
from contextlib import contextmanager
from typing import Generator

from sqlalchemy import Engine, create_engine
from sqlalchemy.ext.asyncio import (AsyncEngine, AsyncSession,
                                    async_sessionmaker, create_async_engine)
from sqlalchemy.orm import Session, sessionmaker

from app.settings import settings

_engine: Engine | None = None
_async_engine: AsyncEngine | None = None


def get_db_engine(db_url: str = settings.db_url) -> Engine:
    global _engine
    if _engine is None:
        _engine = create_engine(db_url)
    return _engine


def get_session() -> sessionmaker[Session]:
    engine = get_db_engine()
    session_factory = sessionmaker(engine, autoflush=False, autocommit=False)
    return session_factory


def get_async_db_engine(db_url: str = settings.db_url) -> AsyncEngine:
    global _async_engine
    if _async_engine is None:
        _async_engine = create_async_engine(db_url)
    return _async_engine


def get_async_session() -> async_sessionmaker[AsyncSession]:
    session_factory = async_sessionmaker(get_async_db_engine())
    return session_factory


@contextmanager
def session_scope():
    """Provide a transactional scope around a series of operations."""
    session = Session()
    try:
        yield session
        session.commit()
    except Exception as exc:
        logging.log(f"SQLAlchemy exception inside session {exc}")
        session.rollback()
        raise
    finally:
        session.close()
