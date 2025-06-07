from litestar.datastructures import UploadFile
from pydantic import BaseModel


class TrackMetadata(BaseModel):
    id: int
    name: str
    artists: str | None
    album: str
    url: str


class UploadMetadata(BaseModel):
    name: str
    artists: str | None
    album: str | None


class TrackStorageId(BaseModel):
    artists: str
    album: str
    name: str

    def get_bucket_name(self) -> str:
        return self.artists

    def get_file_name(self) -> str:
        return f"{self.album}.{self.name}"


class TrackUploadWithMeta(BaseModel):
    name: str
    artists: str | None = None
    album: str | None = None
    file: UploadFile

    class Config:
        arbitrary_types_allowed = True
