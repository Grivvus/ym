from litestar import get


@get("/")
def index() -> str:
    return "Hello world"
