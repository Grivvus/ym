from litestar import Litestar
import uvicorn

import asyncio
import logging

from app.routes import index
from app.settings import settings

app = Litestar([index])


async def main():
    config = uvicorn.Config(
        "main:app", log_level=settings.LOG_LEVEL, host="0.0.0.0"
    )
    server = uvicorn.Server(config)
    logging.info("start uvicorn server")
    await server.serve()


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
