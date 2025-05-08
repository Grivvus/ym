from litestar import Controller, Request, Response, get, patch
from litestar.datastructures import State

from app.database.models import User
from app.services.auth import authorize_by_token


class UserController(Controller):
    path = "/user"

    @get("/get")
    async def get(self, request: Request) -> dict[str, str]:
        username: str = await authorize_by_token(request)
        return {
            "message": f"hello {username}"
        }
