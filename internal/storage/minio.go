package storage

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"github.com/minio/minio-go/v7"
)

type minioStorage struct {
	client *minio.Client
}

func (m minioStorage) PutTrack(
	ctx context.Context, trackID string, r io.Reader,
) error {
	panic("not implemented")
}

func (m minioStorage) GetTrack(
	ctx context.Context, trackID string,
) ([]byte, error) {
	return m.get(ctx, "tracks", trackID, minio.GetObjectOptions{})
}

func (m minioStorage) RemoveTrack(ctx context.Context, trackID string) error {
	panic("not implemented")
}

func (m minioStorage) PutImage(
	ctx context.Context, id string, r io.Reader,
) error {
	panic("not implemented")
}

func (m minioStorage) GetImage(
	ctx context.Context, id string,
) ([]byte, error) {
	return m.get(ctx, "images", id, minio.GetObjectOptions{})
}

func (m minioStorage) ImageExist(ctx context.Context, id string) bool {
	_, err := m.get(ctx, "images", id, minio.GetObjectOptions{})
	return err != nil
}

func (m minioStorage) RemoveImage(ctx context.Context, id string) error {
	err := m.client.RemoveObject(ctx, "images", id, minio.RemoveObjectOptions{})
	return err
}

func (m minioStorage) get(
	ctx context.Context, bucketName, objectID string, opts minio.GetObjectOptions,
) ([]byte, error) {
	obj, err := m.client.GetObject(ctx, bucketName, objectID, opts)
	if err != nil {
		return nil, err
	}
	defer func() { _ = obj.Close() }()
	return io.ReadAll(obj)
}

func (m minioStorage) createBucketsIfNotExists(ctx context.Context) error {
	// this could be a concurrent checks
	// will see on a benchmarks if it's needed
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
			slog.Info("bucket was created:", "bucket", bucketName)
		}
	}
	return nil
}
