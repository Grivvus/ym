import asyncio
import logging

import uvicorn
from litestar import Litestar
from litestar.logging import LoggingConfig

from app.api.http.album import AlbumController
from app.api.http.artist import ArtistController
from app.api.http.auth import AuthController
from app.api.http.track import TrackController
from app.api.http.user import UserController
from app.openapi_configuration import openapi_config
from app.settings import settings

logging_config = LoggingConfig(
    root={"level": "INFO", "handlers": ["queue_listener"]},
    formatters={
        "standard": {
            "format": "%(asctime)s - %(name)s - %(levelname)s - %(message)s",
        }
    },
    log_exceptions="always",
)
app = Litestar(
    route_handlers=[
        AlbumController, ArtistController, AuthController,
        TrackController, UserController,
    ],
    openapi_config=openapi_config,
    logging_config=logging_config
)
app.debug = True


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
