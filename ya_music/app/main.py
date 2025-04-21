import asyncio
import logging

import uvicorn
from litestar import Litestar
from litestar.middleware.base import DefineMiddleware

from app.settings import settings
from app.openapi_configuration import openapi_config
from app.security.authentication_middleware import JWTAuthenticationMiddleware

auth_mw = DefineMiddleware(
    JWTAuthenticationMiddleware, exclide=["schema"],
)

app = Litestar(
    route_handlers=[],
    middleware=[auth_mw],
    openapi_config=openapi_config,
)


async def main():
    config = uvicorn.Config(
        app="main:app",
        log_level=settings.LOG_LEVEL,
        host=settings.APPLICATION_HOST,
        port=settings.APPLICATION_PORT,
    )
    server = uvicorn.Server(config)
    logging.info("start uvicorn server")
    await server.serve()


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
