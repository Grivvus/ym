import logging

from sqlalchemy import select

from app.database.models import Artist
from app.database.utils import get_session


class _ArtistService:
    async def create(self, name: str) -> int:
        with get_session()() as session:
            a = Artist(name=name)
            session.add(a)
            session.commit()
            logging.info(f"artist {name} was created")
            return a.id

    async def is_exist(self, name: str) -> bool:
        stmt = select(Artist)
        with get_session()() as session:
            result = session.execute(stmt)
            fetched = result.scalar_one_or_none()
            if fetched is None:
                return False
            return True

    async def get_by_name(self, name: str) -> Artist | None:
        stmt = select(Artist)
        with get_session()() as session:
            result = session.execute(stmt)
            fetched = result.scalar_one_or_none()
            return fetched

    async def get_id(self, name: str) -> int | None:
        stmt = select(Artist).where(Artist.name == name)
        with get_session()() as session:
            result = session.execute(stmt)
            fetched = result.scalar_one_or_none()
            if fetched is None:
                return None
            return fetched.id


artist_service_provider = _ArtistService()
