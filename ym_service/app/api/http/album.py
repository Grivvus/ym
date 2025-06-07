import logging
from typing import Annotated

from litestar import (Controller, Request, Response, get, patch, post, put,
                      status_codes)

from app.services.album import album_service_provider


class AlbumController(Controller):
    path = "/album"
