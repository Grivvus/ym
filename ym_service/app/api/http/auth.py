from litestar import Controller, Response, post, status_codes
from litestar.exceptions import HTTPException

from app.schemas.user import TokenResponse, UserLogin, UserRegister
from app.services.auth import authenticate_user, register_user


class AuthController(Controller):
    path = "/auth"

    @post("/login", status_code=status_codes.HTTP_200_OK)
    async def login(self, data: UserLogin) -> TokenResponse:
        if not data.username or not data.password:
            raise HTTPException(
                status_code=status_codes.HTTP_400_BAD_REQUEST,
                detail="username and password are requires",
            )
        user, token = await authenticate_user(data)
        return TokenResponse(
            id=user.id, username=user.username, email=user.email,
            token_type="bearer", access_token=token,
        )

    @post("/register")
    async def register(self, data: UserRegister) -> TokenResponse:
        if not data.username or not data.password:
            raise HTTPException(
                status_code=status_codes.HTTP_400_BAD_REQUEST,
                detail="username, password are requires",
            )

        user, token = await register_user(data)
        return TokenResponse(
            id=user.id, username=user.username, email=user.email,
            token_type="bearer", access_token=token,
        )
