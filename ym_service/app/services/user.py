import logging

from litestar import status_codes
from litestar.exceptions import (ClientException, HTTPException,
                                 NotAuthorizedException)
from pydantic import EmailStr
from sqlalchemy import select, update
from sqlalchemy.exc import IntegrityError

from app.database.models import User
from app.database.utils import get_session
from app.schemas.user import UserChangePassword
from app.security.jwt import hash_password, verify_password


class _UserService:
    async def change_user_password(self, data: UserChangePassword) -> None:
        if len(data.new_password.get_secret_value()) < 6:
            raise ValueError("password should be 6 symbols or more")
        stmt = select(User).where(User.username == data.username)
        fetched_user: User | None = None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_user = result.scalar_one_or_none()

        if fetched_user is None:
            raise NotAuthorizedException("wrong username")
        if not verify_password(
            data.current_password.get_secret_value(), fetched_user.password
        ):
            raise NotAuthorizedException("wrong password")

        with get_session()() as session:
            fetched_user.password = hash_password(
                data.new_password.get_secret_value()
            )
            session.add(fetched_user)
            session.commit()
            logging.info(f"password of user {data.username} has change")

    async def change_email(self, username: str, new_email: EmailStr) -> None:
        stmt = select(User).where(User.username == username)
        fetched_user: User | None = None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_user = result.scalar_one_or_none()
        if fetched_user is None:
            raise NotAuthorizedException("wrong username")
        try:
            with get_session()() as session:
                fetched_user.email = new_email
                session.add(fetched_user)
                session.commit()
                logging.info(f"User {username} change email")
        except IntegrityError as e:
            logging.warning(e)
            raise HTTPException(
                "this email is used already",
                status_codes.HTTP_400_BAD_REQUEST
            )

    async def change_username(
        self, username: str, new_username: str
    ) -> None:
        stmt = select(User).where(User.username == username)
        fetched_user: User | None = None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_user = result.scalar_one_or_none()
        if fetched_user is None:
            raise NotAuthorizedException(f"wrong username {username}")
        try:
            with get_session()() as session:
                fetched_user.username = new_username
                session.add(fetched_user)
                session.commit()
                logging.info(f"User {username} change email")
        except IntegrityError as e:
            logging.warning(e)
            raise HTTPException(
                "New username is used already",
                status_codes.HTTP_400_BAD_REQUEST
            )

    async def get_user_id(self, username: str) -> int:
        stmt = select(User).where(User.username == username)
        fetched_user: User | None = None
        with get_session()() as session:
            result = session.execute(stmt)
            fetched_user = result.scalar_one_or_none()
        if fetched_user is None:
            raise NotAuthorizedException("wrong username")
        return fetched_user.id


user_service_provider = _UserService()
