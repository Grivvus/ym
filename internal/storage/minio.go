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
	ctx context.Context, id string, r io.Reader,
) error {
	return m.put(ctx, "tracks", id, r, minio.PutObjectOptions{})
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
	return m.put(ctx, "images", id, r, minio.PutObjectOptions{})
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
	return err != nil
}

func (m minioStorage) RemoveImage(ctx context.Context, id string) error {
	return m.remove(ctx, "images", id, minio.RemoveObjectOptions{})
}

func (m minioStorage) get(
	ctx context.Context, bucketName, objectID string, opts minio.GetObjectOptions,
) (io.ReadSeekCloser, error) {
	return m.client.GetObject(ctx, bucketName, objectID, opts)
}

func (m minioStorage) put(
	ctx context.Context, bucketName, objectID string,
	r io.Reader, opts minio.PutObjectOptions,
) error {
	uinfo, err := m.client.PutObject(ctx, bucketName, objectID, r, -1, opts)
	_ = uinfo
	return err
}

func (m minioStorage) remove(
	ctx context.Context,
	bucketName, id string,
	opts minio.RemoveObjectOptions,
) error {
	return m.client.RemoveObject(ctx, bucketName, id, opts)
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
