from litestar import Request, status_codes
from litestar.exceptions import HTTPException, NotAuthorizedException
from sqlalchemy import select
from sqlalchemy.exc import IntegrityError

from app.database.models import User
from app.database.utils import get_session
from app.schemas.user import UserLogin, UserRegister
from app.security.jwt import Token, decode_jwt_token, encode_jwt_token


async def authenticate_user(user: UserLogin) -> str:
    stmt = select(User).where(User.username == user.username)
    fetched_user: User | None = None
    with get_session()() as session:
        result = session.execute(stmt)
        fetched_user = result.scalar_one_or_none()

    if fetched_user is None:
        raise HTTPException(
            status_codes.HTTP_401_UNAUTHORIZED,
            detail="wrong username"
        )
    return encode_jwt_token(fetched_user.username)


async def register_user(user: UserRegister) -> str:
    new_user = User(
        username=user.username,
        email=user.email,
        password=user.password
    )
    with get_session()() as session:
        try:
            session.add(new_user)
            session.commit()
        except IntegrityError:
            raise HTTPException(
                status_codes.HTTP_400_BAD_REQUEST,
                detail="username is not unique"
            )

    return encode_jwt_token(new_user.username)


async def authorize_by_token(request: Request) -> Token:
    auth_header = request.headers.get("Authorizetion")
    if auth_header is None:
        raise HTTPException(
            status_codes.HTTP_401_UNAUTHORIZED,
            detail="not authenticated",
        )
    print(auth_header)
    try:
        payload = decode_jwt_token(auth_header)
        return payload
    except NotAuthorizedException as err:
        raise HTTPException(
            status_codes.HTTP_401_UNAUTHORIZED,
            detail=f"err {err}"
        )
