from litestar import Request, status_codes
from litestar.exceptions import HTTPException, NotAuthorizedException
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError

from app.database.models import User
from app.database.utils import get_session
from app.schemas.user import UserLogin, UserRegister
from app.security.jwt import (decode_jwt_token, encode_jwt_token,
                              hash_password, verify_password)


async def authenticate_user(user: UserLogin) -> str:
    stmt = select(User).where(User.username == user.username)
    fetched_user: User | None = None
    with get_session()() as session:
        result = session.execute(stmt)
        fetched_user = result.scalar_one_or_none()

    if fetched_user is None:
        raise NotAuthorizedException("wrong username")
    if not verify_password(
        user.password.get_secret_value(), fetched_user.password
    ):
        raise NotAuthorizedException("wrong password")
    return encode_jwt_token(fetched_user.username)


async def register_user(user: UserRegister) -> str:
    with get_session().begin() as session:
        try:
            new_user = User(
                username=user.username,
                email=user.email,
                password=hash_password(user.password.get_secret_value())
            )
            session.add(new_user)
            session.commit()
        except IntegrityError:
            raise HTTPException(
                status_codes.HTTP_400_BAD_REQUEST,
                detail="username is not unique"
            )
    return encode_jwt_token(user.username)


async def authorize_by_token(request: Request) -> str:
    auth_header = request.headers.get("authorization")
    if auth_header is None:
        raise NotAuthorizedException("missing 'authorization' header")
    username = decode_jwt_token(auth_header)
    return username
