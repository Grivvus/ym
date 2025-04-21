from datetime import datetime, timedelta
from uuid import UUID

from jose import JWTError, jwt
from pydantic import UUID4, BaseModel

from app.settings import settings
from litestar.exceptions import NotAuthorizedException

DEFAULT_TIME_DELTA = timedelta(days=1)
ALGORITHM = "HS256"


class Token(BaseModel):
    exp: datetime
    iat: datetime
    sub: UUID4


def decode_jwt_token(encoded_token: str) -> Token:
    try:
        payload = jwt.decode(
            token=encoded_token,
            key=settings.JWT_SECRET,
            algorithms=[ALGORITHM]
        )
        return Token(**payload)
    except JWTError as e:
        raise NotAuthorizedException("Invalid token") from e


def encode_jwt_token(
    user_id: UUID, expiration: timedelta = DEFAULT_TIME_DELTA
) -> str:
    token = Token(
        exp=datetime.now() + expiration,
        iat=datetime.now(),
        sub=user_id,
    )
    return jwt.encode(token.dict(), settings.JWT_SECRET, algorithm=ALGORITHM)
