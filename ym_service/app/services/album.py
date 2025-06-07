import logging

from sqlalchemy import select

from app.database.models import Album
from app.database.utils import get_session
from app.services.artist import artist_service_provider


class _AlbumService:
    async def create(self, name: str, artist: str | int) -> int:
        if isinstance(artist, str):
            fetched_artist = await artist_service_provider.get_by_name(artist)
            if fetched_artist is None:
                artist_id = await artist_service_provider.create(artist)
            else:
                artist_id = fetched_artist.id
        else:
            artist_id = artist
        with get_session()() as session:
            a = Album(name=name, artist_id=artist_id)
            session.add(a)
            session.commit()
            logging.info(f"album {name} was created")
            return a.id

    async def get_id(self, name: str, artist_id: int) -> int | None:
        stmt = select(Album).where(
            Album.artist_id == artist_id and Album.name == name
        )
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_album = result.scalar_one_or_none()
            if fetched_album is None:
                return None
            return fetched_album.id


album_service_provider = _AlbumService()
