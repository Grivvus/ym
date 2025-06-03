import logging
from io import BytesIO

import minio
from litestar import status_codes
from litestar.datastructures import UploadFile
from litestar.exceptions import HTTPException

from app.settings import settings


def get_storage_client() -> minio.Minio:
    return minio.Minio(
        endpoint=settings.S3_HOST + ":" + settings.S3_PORT,
        access_key=settings.S3_ROOT_USER,
        secret_key=settings.S3_ROOT_PASSWORD,
        cert_check=False, secure=False
    )


async def upload_image(bucket_name: str, file: UploadFile) -> None:
    logging.warning(
        "bad realization of upload_image, possible slow and block flow"
    )
    storage = get_storage_client()

    if not storage.bucket_exists(bucket_name):
        storage.make_bucket(bucket_name)

    bytes_ = await file.read()
    file_data = BytesIO(bytes_)

    try:
        res = storage.put_object(
            bucket_name,
            file.filename,
            file_data,
            len(bytes_),
        )
        logging.info(
            f"file {file.filename} uploaded successfully with res {res}"
        )
    except minio.error.S3Error as e:
        logging.error(f"Error while uploading file {e}")
        raise HTTPException(status_codes.HTTP_503_SERVICE_UNAVAILABLE)


async def download_image(bucket_name: str, filename: str) -> BytesIO | None:
    storage = get_storage_client()

    if not storage.bucket_exists(bucket_name):
        storage.make_bucket(bucket_name)
        return None

    try:
        response = storage.get_object(bucket_name, filename)
        data = response.data
        return BytesIO(data)
    except minio.error.S3Error as e:
        logging.error(f"Error while downloading file {e}")
        raise HTTPException(status_codes.HTTP_503_SERVICE_UNAVAILABLE)
    finally:
        response.close()
        response.release_conn()
