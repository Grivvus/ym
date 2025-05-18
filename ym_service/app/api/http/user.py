from typing import Annotated

from litestar import Controller, Request, get, post
from litestar.datastructures import UploadFile
from litestar.enums import RequestEncodingType
from litestar.params import Body

from app.database.storage import upload_image
from app.services.auth import authorize_by_token


class UserController(Controller):
    path = "/user"

    @get("/get")
    async def get(self, request: Request) -> dict[str, str]:
        username: str = await authorize_by_token(request)
        return {
            "message": f"hello {username}"
        }

    @post("/upload_avatar")
    async def upload_avatar(
        self, username: str,
        data: Annotated[
            UploadFile, Body(media_type=RequestEncodingType.MULTI_PART)
        ],
    ) -> None:
        data.filename = "avatar"
        await upload_image(username, data)
