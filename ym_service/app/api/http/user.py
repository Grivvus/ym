import logging
from typing import Annotated

from litestar import Controller, Request, get, patch, post, put, status_codes
from litestar.datastructures import UploadFile
from litestar.enums import RequestEncodingType
from litestar.exceptions import HTTPException, NotAuthorizedException
from litestar.params import Body
from litestar.response import File

from app.database.storage import upload_image
from app.schemas.user import UserChangePassword
from app.services.auth import authorize_by_token
from app.services.user import user_service_provider


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

    # @get("/avatar", media_type="image/png")
    # async def get_avatar(self, username: str) -> File:
    #     ...

    @patch("/change_password")
    async def change_password(
        self, data: UserChangePassword
    ) -> None:
        try:
            await user_service_provider.change_user_password(data)
        except ValueError as e:
            raise HTTPException(
                e, status_codes.HTTP_400_BAD_REQUEST,
            )
        except NotAuthorizedException as e:
            logging.error(f"Caght an error: {e}")
            raise HTTPException(
                "Can't change password, wrong username or old password",
                status_code=status_codes.HTTP_401_UNAUTHORIZED,
            )
        except Exception as e:
            logging.error(f"Caght an error: {e}")
            raise HTTPException(
                "Enternal server error",
                status_codes.HTTP_500_INTERNAL_SERVER_ERROR,
            )
