import logging
from io import BytesIO
from typing import Sequence

from litestar import status_codes
from litestar.datastructures import UploadFile
from litestar.exceptions import (ClientException, HTTPException,
                                 NotAuthorizedException)
from pydantic import EmailStr
from sqlalchemy import select, update
from sqlalchemy.exc import IntegrityError
from sqlalchemy.orm import joinedload, selectinload

from app.database import storage
from app.database.models import Track
from app.database.utils import get_session
from app.schemas.track import TrackMetadata, UploadMetadata
from app.services.album import album_service_provider
from app.services.artist import artist_service_provider


class _TrackService:
    async def get_track_metadata(self, track_url: str) -> TrackMetadata:
        ...

    async def get_track_by_id(self, id: int) -> TrackMetadata | None:
        stmt = select(Track).where(Track.id == id).options(
            selectinload(Track.artist),
            selectinload(Track.album)
        )
        fetched_track: Track | None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_track = result.scalar_one_or_none()
        if fetched_track is None:
            return None
        if not fetched_track.artist:
            logging.error("artists is empty")
            raise RuntimeError("track should has at least 1 artist")
        return TrackMetadata(
            id=fetched_track.id, name=fetched_track.name,
            artist=str(fetched_track.artist),
            album=str(fetched_track.album),
            url=fetched_track.url,
        )

    async def get_raw_track_by_id(self, id: int) -> Track:
        stmt = select(Track).where(Track.id == id).options(
            selectinload(Track.artist),
            selectinload(Track.album)
        )
        fetched_track: Track | None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_track = result.scalar_one_or_none()
        if fetched_track is None:
            raise ValueError("No suck track")
        return fetched_track

    async def get_initial_tracks(self) -> list[TrackMetadata]:
        stmt = (
            select(Track)
            .options(
                selectinload(Track.artist),
                selectinload(Track.album)
            )
        )
        fetched_tracks: Sequence[Track]
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_tracks = result.scalars().unique().all()
        ret = []
        for ft in fetched_tracks:
            ret.append(TrackMetadata(
                id=ft.id, name=ft.name,
                artist=str(ft.artist),
                album=str(ft.album),
                url=f"{str(ft.album)}.{ft.name}",
            ))
        return ret

    async def upload_track(
        self, upload_metadata: UploadMetadata,
        is_uploaded_by_user: bool,
        data: UploadFile,
    ) -> int:
        artist_id: int | None
        album_id: int | None

        if upload_metadata.artist is None:
            upload_metadata.artist = "unkown"
        if upload_metadata.album is None:
            upload_metadata.album = "unkown"
        artist_id = await artist_service_provider.get_id(
            upload_metadata.artist
        )
        if artist_id is None:
            logging.info("artist didn't exist branch")
            artist_id = await artist_service_provider.create(
                upload_metadata.artist
            )
            logging.info("creating album")
            album_id = await album_service_provider.create(
                upload_metadata.album, artist_id
            )
        else:
            logging.info("artist exist, checking for album")
            album_id = await album_service_provider.get_id(
                upload_metadata.album, artist_id
            )
        if album_id is None:
            logging.info("album didn't exist")
            album_id = await album_service_provider.create(
                upload_metadata.album, artist_id
            )

        url_str = f"{upload_metadata.album.lower()}.{upload_metadata.name.lower()}"
        with get_session()() as session:
            track = Track(
                name=upload_metadata.name,
                duration=-1,
                is_uploaded_by_user=is_uploaded_by_user,
                url=url_str,
                album_id=album_id,
                artist_id=artist_id,
            )
            session.add(track)
            session.commit()
            await storage.upload_track(
                f"{artist_id}-{album_id}", upload_metadata.name, data
            )
            return track.id

    async def download_track(self, track_id: int) -> BytesIO:
        track = await self.get_raw_track_by_id(track_id)
        data = await storage.download_track(
            f"{track.artist_id}-{track.album_id}", track.name
        )
        return data


track_service_provider = _TrackService()
