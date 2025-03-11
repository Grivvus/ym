from sqlalchemy import (
    Interger, String
)
from sqlalchemy.orm import (
    DeclarativeBase, mapped_column
)


class Base(DeclarativeBase):
    pass


class User(Base):
    __tablename__ = "user"


class Artist(Base):
    __tablename__ = "artist"


class Track(Base):
    __tablename__ = "track"


class Album(Base):
    __tablename__ = "album"


class Playlist(Base):
    __tablename__ = "playlist"
