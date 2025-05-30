import logging

from litestar.exceptions import HTTPException, NotAuthorizedException
from sqlalchemy import select, update

from app.database.models import User
from app.database.utils import get_session
from app.schemas.user import UserChangePassword
from app.security.jwt import hash_password, verify_password


class _UserService:
    async def change_user_password(self, data: UserChangePassword):
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


user_service_provider = _UserService()
