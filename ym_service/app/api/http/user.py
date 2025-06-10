import logging
from typing import Annotated

from litestar import (Controller, Request, Response, get, patch, post, put,
                      status_codes)
from litestar.datastructures import UploadFile
from litestar.enums import RequestEncodingType
from litestar.exceptions import HTTPException, NotAuthorizedException
from litestar.params import Body
from litestar.response.streaming import Stream

from app.database.storage import download_image, upload_image
from app.schemas.user import BasicUserSchema, UserChange, UserChangePassword
from app.services.auth import authorize_by_token
from app.services.user import user_service_provider


class UserController(Controller):
    path = "/user"

    @get("/{id:int}")
    async def get_user(self, id: int) -> BasicUserSchema:
        user = await user_service_provider.get_user_by_id(id)
        return BasicUserSchema(
            id=user.id,
            username=user.username,
            email=user.email,
        )

    @post("/avatar/{username:str}")
    async def upload_avatar(
        self, username: str,
        data: Annotated[
            UploadFile, Body(media_type=RequestEncodingType.MULTI_PART)
        ],
    ) -> None:
        user = await user_service_provider.get_user(username)
        await upload_image("avatars", f"{user.id}-user", data)

    @get("/avatar/{username:str}", media_type="image/*")
    async def download_avatar(self, username: str) -> Stream | None:
        user = await user_service_provider.get_user(username)
        data = await download_image("avatars", f"{user.id}-user")
        if data is None:
            return None
        return Stream(content=data, media_type="image/*")

    @patch("/")
    async def change(self, data: UserChange) -> None:
        if data.new_username is not None:
            try:
                await user_service_provider.change_username(
                    data.username, data.new_username
                )
            except NotAuthorizedException as e:
                logging.warning(f"No permission to do this {e}")
                raise HTTPException(
                    f"No permission to do this {e}",
                    status_codes.HTTP_401_UNAUTHORIZED
                )
        if data.new_email is not None:
            try:
                await user_service_provider.change_email(
                    data.username,
                    data.new_email,
                )
            except NotAuthorizedException as e:
                logging.warning(
                    f"User {data.username} can't change email, exc: {e}"
                )
                raise HTTPException(
                    "Can't change email; this email is used",
                    status_codes.HTTP_400_BAD_REQUEST,
                )

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
