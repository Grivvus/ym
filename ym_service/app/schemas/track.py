from litestar.datastructures import UploadFile
from pydantic import BaseModel


class TrackMetadata(BaseModel):
    id: int
    name: str
    artist: str | None
    album: str
    url: str


class UploadMetadata(BaseModel):
    name: str
    artist: str | None
    album: str | None


class TrackStorageId(BaseModel):
    artist: str
    album: str
    name: str

    def get_bucket_name(self) -> str:
        return self.artist

    def get_file_name(self) -> str:
        return f"{self.album}.{self.name}"


class TrackUploadWithMeta(BaseModel):
    name: str
    artist: str | None = None
    album: str | None = None
    file: UploadFile

    class Config:
        arbitrary_types_allowed = True
