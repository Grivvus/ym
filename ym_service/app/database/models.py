from __future__ import annotations

from datetime import datetime

from sqlalchemy import (Boolean, Column, ForeignKey, Integer, String, Table,
                        func)
from sqlalchemy.orm import DeclarativeBase, Mapped, mapped_column, relationship


class Base(DeclarativeBase):
    pass


track_playlist = Table(
    "track_playlist",
    Base.metadata,
    Column("track_id", ForeignKey("track.id"), primary_key=True),
    Column("playlist_id", ForeignKey("playlist.id"), primary_key=True),
)

artist_track = Table(
    "artist_track",
    Base.metadata,
    Column("artist_id", ForeignKey("artist.id"), primary_key=True),
    Column("track_id", ForeignKey("track.id"), primary_key=True),
)


class User(Base):
    __tablename__ = "user"

    id: Mapped[int] = mapped_column(primary_key=True)
    username: Mapped[str] = mapped_column(String(50), unique=True)
    email: Mapped[str | None] = mapped_column(String(318), unique=True)  # 64 + @ + 253
    password: Mapped[str] = mapped_column(String)
    created_at: Mapped[datetime] = mapped_column(insert_default=func.now())
    updated_at: Mapped[datetime] = mapped_column(insert_default=func.now())

    playlists_owned: Mapped[list[Playlist]] = relationship(
        back_populates="owner"
    )


class Artist(Base):
    __tablename__ = "artist"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String)

    albums: Mapped[list[Album]] = relationship(back_populates="artist")
    tracks: Mapped[list[Track]] = relationship(
        secondary=artist_track, back_populates="artists"
    )


class Track(Base):
    __tablename__ = "track"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String)
    duration: Mapped[int] = mapped_column(Integer)
    is_uploaded_by_user: Mapped[bool] = mapped_column(
        Boolean(create_constraint=True)
    )

    artists: Mapped[list[Artist]] = relationship(
        secondary=artist_track, back_populates="tracks"
    )


class Album(Base):
    __tablename__ = "album"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String)

    artist_id: Mapped[int] = mapped_column(ForeignKey("artist.id"))
    artist: Mapped[Artist] = relationship(back_populates="albums")


class Playlist(Base):
    __tablename__ = "playlist"

    id: Mapped[int] = mapped_column(primary_key=True)
    name: Mapped[str] = mapped_column(String)

    owner_id: Mapped[int | None] = mapped_column(ForeignKey("user.id"))
    owner: Mapped[User | None] = relationship(back_populates="playlist_owned")
    tracks: Mapped[list[Track]] = relationship(secondary=track_playlist)
