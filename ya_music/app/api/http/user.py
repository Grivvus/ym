from litestar import Request, Response, get, post
from litestar.datastructures import State

from app.security.jwt import Token
from app.database.models import User


@get("/")
def index() -> str:
    return "Hello world"


@post("/login")
def login(request: Request[User, Token, State]) -> None:
    user = request.user
    auth = request.auth
    assert isinstance(user, User)
    assert isinstance(auth, Token)
