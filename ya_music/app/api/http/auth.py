from litestar import Controller, get, post, status_codes
from litestar.exceptions import HTTPException

from app.schemas.user import UserLogin, UserRegister
from app.services.auth import authenticate_user, register_user


class AuthController(Controller):
    path = "/auth"

    @post("/login")
    async def login(self, data: UserLogin) -> dict[str, str]:
        if not data.username or not data.password:
            raise HTTPException(
                status_code=status_codes.HTTP_400_BAD_REQUEST,
                detail="username and password are requires",
            )
        token = await authenticate_user(data)
        return {"access_token": token, "token_type": "bearer"}

    @post("/register")
    async def register(self, data: UserRegister) -> dict[str, str]:
        if not data.username or not data.email or not data.password:
            raise HTTPException(
                status_code=status_codes.HTTP_400_BAD_REQUEST,
                detail="username, email and password are requires",
            )

        token = await register_user(data)
        return {"access_token": token, "token_type": "bearer"}
