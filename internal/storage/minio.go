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

func (m minioStorage) PutCover(
	ctx context.Context, coverID string, r io.Reader,
) error {
	panic("not implemented")
}

func (m minioStorage) GetCover(
	ctx context.Context, coverID string,
) ([]byte, error) {
	return m.get(ctx, "covers", coverID, minio.GetObjectOptions{})
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
	for _, bucketName := range []string{"covers", "tracks"} {
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
