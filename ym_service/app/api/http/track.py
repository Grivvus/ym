import logging
from typing import Annotated

from litestar import (Controller, Request, Response, get, patch, post, put,
                      status_codes)
from litestar.datastructures import UploadFile
from litestar.enums import RequestEncodingType
from litestar.exceptions import (ClientException, HTTPException,
                                 NotAuthorizedException)
from litestar.params import Body
from litestar.response.streaming import Stream

from app.database.storage import download_track
from app.schemas.track import (TrackMetadata, TrackStorageId,
                               TrackUploadWithMeta, UploadMetadata)
from app.services.track import track_service_provider


class TrackController(Controller):
    path = "/track"

    @get("/get_initial")
    async def get_initial(self) -> list[TrackMetadata]:
        data = await track_service_provider.get_initial_tracks()
        return data

    @post("/upload")
    async def upload_track(
        self,
        data: TrackUploadWithMeta = Body(
            media_type=RequestEncodingType.MULTI_PART
        ),
    ) -> int:
        ret_id = await track_service_provider.upload_track(
            UploadMetadata(
                name=data.name,
                artist=data.artist,
                album=data.album,
            ), True, data.file
        )
        return ret_id

    @get("/{track_id:int}")
    async def get_track_by_id(self, track_id: int) -> Stream | None:
        data = await track_service_provider.download_track(track_id)
        if data is None:
            return None
        return Stream(content=data, media_type="audio/mpeg")

    @get("/track_meta/{id:int}")
    async def get_track_meta_by_id(self, id: int) -> TrackMetadata:
        tm = await track_service_provider.get_track_by_id(id)
        if tm is None:
            raise HTTPException(
                "Wrong track id",
                status_codes.HTTP_404_NOT_FOUND,
            )
        return tm
