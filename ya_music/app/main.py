from litestar import Litestar
import uvicorn

import asyncio
import logging


app = Litestar()


async def main():
    config = uvicorn.Config("main:app", log_level="debug")
    server = uvicorn.Server(config)
    logging.info("start uvicorn server")
    await server.serve()


if __name__ == "__main__":
    asyncio.run(main(), debug=True)
