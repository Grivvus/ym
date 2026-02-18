package storage

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Grivvus/ym/internal/utils"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// The type represents an implementation-independent representation of object storage.
// All implementations (Minio, file-system) will respect this interface.
type Storage interface {
	/* may return some other type, represents info */
	PutTrack(ctx context.Context, trackID string, r io.Reader) error
	// may return some type, that represents more info about object and implements io.Reader
	GetTrack(ctx context.Context, trackID string) (io.ReadSeekCloser, error)
	GetTrackInfo(ctx context.Context, trackID string) (clen uint, ctype string, err error)
	RemoveTrack(ctx context.Context, trackID string) error

	PutImage(ctx context.Context, imageID string, r io.Reader) error
	GetImage(ctx context.Context, imageID string) ([]byte, error)
	RemoveImage(ctx context.Context, imageID string) error
	ImageExist(ctx context.Context, imageID string) bool
}

func New(cfg utils.Config) (Storage, error) {
	if cfg.S3Host == "" {
		return nil, errors.New("can't create storage, S3_HOST env variable is not set")
	}
	s3URL := cfg.S3Host + ":" + cfg.S3Port
	minioClient, err := minio.New(
		s3URL,
		&minio.Options{
			Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
			Secure: false,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("can't create connection to minio: %w", err)
	}
	storage := minioStorage{client: minioClient}
	err = storage.createBucketsIfNotExists(context.TODO())
	if err != nil {
		return nil, fmt.Errorf("can't init storage: %w", err)
	}
	return storage, nil
}
