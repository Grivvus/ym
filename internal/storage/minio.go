package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

type minioStorage struct {
	client *minio.Client
	logger *slog.Logger
}

func (m minioStorage) PutTrack(
	ctx context.Context, id string, r io.Reader, objSize int64,
) error {
	return m.put(ctx, "tracks", id, objSize, r, minio.PutObjectOptions{})
}

func (m minioStorage) GetTrackInfo(
	ctx context.Context, id string,
) (uint, string, error) {
	info, err := m.client.StatObject(ctx, "tracks", id, minio.StatObjectOptions{})
	if err != nil {
		return 0, "", err
	}
	return uint(info.Size), info.ContentType, nil
}

func (m minioStorage) GetTrack(
	ctx context.Context, id string,
) (io.ReadSeekCloser, error) {
	return m.get(ctx, "tracks", id, minio.GetObjectOptions{})
}

func (m minioStorage) RemoveTrack(ctx context.Context, trackID string) error {
	return m.remove(ctx, "tracks", trackID, minio.RemoveObjectOptions{})
}

func (m minioStorage) PutImage(
	ctx context.Context, id string, r io.Reader,
) error {
	return m.put(ctx, "images", id, -1, r, minio.PutObjectOptions{})
}

func (m minioStorage) GetImage(
	ctx context.Context, id string,
) ([]byte, error) {
	rsc, err := m.get(ctx, "images", id, minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer func() { _ = rsc.Close() }()
	return io.ReadAll(rsc)
}

func (m minioStorage) ImageExist(ctx context.Context, id string) bool {
	_, err := m.get(ctx, "images", id, minio.GetObjectOptions{})
	if err == nil {
		return false
	}
	if errResp, ok := errors.AsType[minio.ErrorResponse](err); ok {
		return errResp.Code != minio.NoSuchKey
	}
	return false
}

func (m minioStorage) RemoveImage(ctx context.Context, id string) error {
	return m.remove(ctx, "images", id, minio.RemoveObjectOptions{})
}

func (m minioStorage) get(
	ctx context.Context, bucketName, objectID string, opts minio.GetObjectOptions,
) (io.ReadSeekCloser, error) {
	obj, err := m.client.GetObject(ctx, bucketName, objectID, opts)
	if err != nil {
		if e, ok := errors.AsType[minio.ErrorResponse](err); ok && e.Code == minio.NoSuchKey {
			return nil, fmt.Errorf("%w: no such key %v", ErrObjectNotFound, objectID)
		}
		return nil, fmt.Errorf("%w caused by: %w", InternalStorageError, err)
	}
	return obj, nil
}

func (m minioStorage) put(
	ctx context.Context, bucketName, objectID string, objSize int64,
	r io.Reader, opts minio.PutObjectOptions,
) error {
	_, err := m.client.PutObject(ctx, bucketName, objectID, r, objSize, opts)
	if err != nil {
		if e, ok := errors.AsType[minio.ErrorResponse](err); ok {
			if e.Code == minio.EntityTooSmall || e.Code == minio.EntityTooLarge {
				return fmt.Errorf(
					"%w: size of the object is bad, %w",
					ErrBadObject, err,
				)
			}
			return fmt.Errorf("%w caused by: %w", InternalStorageError, err)
		}
		return fmt.Errorf("%w caused by: %w", InternalStorageError, err)
	}
	return nil
}

func (m minioStorage) remove(
	ctx context.Context,
	bucketName, id string,
	opts minio.RemoveObjectOptions,
) error {
	err := m.client.RemoveObject(ctx, bucketName, id, opts)
	if err != nil {
		if e, ok := errors.AsType[minio.ErrorResponse](err); ok && e.Code == minio.NoSuchKey {
			return fmt.Errorf("%w: no such key %v", ErrObjectNotFound, id)
		}
		return fmt.Errorf("%w caused by: %w", InternalStorageError, err)
	}
	return nil
}

func (m minioStorage) createBucketsIfNotExists(ctx context.Context) error {
	for _, bucketName := range []string{"images", "tracks"} {
		found, err := m.client.BucketExists(ctx, bucketName)
		if err != nil {
			return err
		}
		if !found {
			err := m.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
			if err != nil {
				return fmt.Errorf("can't create new bucket: %w", err)
			}
			m.logger.Info("bucket was created:", "bucket", bucketName)
		}
	}
	return nil
}
