from datetime import datetime

from sqlalchemy import String, func
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column


class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(50), unique=True)
    email: Mapped[str | None] = mapped_column(String(318))  # 64 + @ + 253
    password: Mapped[str] = mapped_column(String)
    created_at: Mapped[datetime] = mapped_column(insert_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(insert_default=func.now())


# class Artist(Base):
#     __tablename__ = "artist"
#
#
# class Track(Base):
#     __tablename__ = "track"
#
#
# class Album(Base):
#     __tablename__ = "album"
#
#
# class Playlist(Base):
#     __tablename__ = "playlist"
