import logging
from typing import Annotated

from litestar import (Controller, Request, Response, get, patch, post, put,
                      status_codes)

from app.services.artist import artist_service_provider


class ArtistController(Controller):
    path = "/artist"
