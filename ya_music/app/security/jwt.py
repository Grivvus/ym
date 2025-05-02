from datetime import datetime, timedelta

from jose import JWTError, jwt
from litestar.exceptions import NotAuthorizedException
from pydantic import BaseModel

from app.settings import settings

DEFAULT_TIME_DELTA = timedelta(days=1)
ALGORITHM = "HS256"


class Token(BaseModel):
    expiration: datetime
    issued_at: datetime
    username: str


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
    username: str, expiration: timedelta = DEFAULT_TIME_DELTA
) -> str:
    token = Token(
        expiration=datetime.now() + expiration,
        issued_at=datetime.now(),
        username=username,
    )
    return jwt.encode(token.dict(), settings.JWT_SECRET, algorithm=ALGORITHM)
