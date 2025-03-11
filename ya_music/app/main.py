import asyncio
import logging

import uvicorn
from litestar import Litestar

from app.settings import settings
from app.configuration import openapi_config

app = Litestar(
    route_handlers=[],
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
