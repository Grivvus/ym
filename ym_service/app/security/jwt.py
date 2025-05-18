from datetime import datetime, timedelta, timezone

from litestar.security.jwt import Token
from passlib.context import CryptContext

from app.settings import settings

DEFAULT_TIME_DELTA = timedelta(days=1)
ALGORITHM = "HS256"
pwd_context = CryptContext(["bcrypt"], deprecated="auto")


def decode_jwt_token(encoded_token: str) -> str:
    payload = Token.decode(encoded_token, settings.JWT_SECRET, ALGORITHM)
    return payload.sub


def encode_jwt_token(
    username: str, expires_delta: timedelta = DEFAULT_TIME_DELTA
) -> str:
    t = Token(datetime.now(timezone.utc) + expires_delta, username)
    return t.encode(settings.JWT_SECRET, ALGORITHM)


def verify_password(plain_password: str, hashed: str) -> bool:
    return pwd_context.verify(plain_password, hashed)


def hash_password(plain: str) -> str:
    return pwd_context.hash(plain)
