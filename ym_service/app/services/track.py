import logging
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
            selectinload(Track.artists),
            selectinload(Track.album)
        )
        fetched_track: Track | None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_track = result.scalar_one_or_none()
        if fetched_track is None:
            return None
        if not fetched_track.artists:
            logging.warning("artists is empty")
            artists_str = "anamnez"
        else:
            artists_str = ", ".join(
                [str(artist) for artist in fetched_track.artists]
            )[:-2]
        print(str(fetched_track.album))
        print(str(fetched_track.artists))
        return TrackMetadata(
            id=fetched_track.id, name=fetched_track.name,
            artists=artists_str, album=str(fetched_track.album),
            url=fetched_track.url,
        )

    async def get_initial_tracks(self) -> list[TrackMetadata]:
        stmt = (
            select(Track)
            .options(
                selectinload(Track.artists),
                selectinload(Track.album)
            )
        )
        fetched_tracks: Sequence[Track]
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_tracks = result.scalars().unique().all()
        ret = []
        for ft in fetched_tracks:
            artsts_str = ", ".join(str(ft.artists))[:-2]
            ret.append(TrackMetadata(
                id=ft.id, name=ft.name, artists=artsts_str,
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

        if upload_metadata.artists is None:
            upload_metadata.artists = "unkown"
        if upload_metadata.album is None:
            upload_metadata.album = "unkown"
        if upload_metadata.artists.count(","):
            artist_id = await artist_service_provider.get_id(
                upload_metadata.artists.split(",")[0]
            )
        else:
            artist_id = await artist_service_provider.get_id(
                upload_metadata.artists
            )
        if artist_id is None:
            logging.info("artist didn't exist branch")
            artist_id = await artist_service_provider.create(
                upload_metadata.artists
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
            )
            session.add(track)
            session.commit()
            await storage.upload_track(upload_metadata.artists, url_str, data)
            return track.id


track_service_provider = _TrackService()
