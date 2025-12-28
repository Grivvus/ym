package storage

import (
	"context"
	"io"
)

// The type represents an implementation-independent representation of object storage.
// All implementations (Minio, file-system) will respect this interface.
type Storage interface {
	/*may return some other type, represents info*/
	PutTrack(ctx context.Context, trackID string, r io.Reader) error
	// may return some type, that represents more info about object and implements io.Reader
	GetTrack(ctx context.Context, trackID string) ([]byte, error)

	PutCover(ctx context.Context, coverID string, r io.Reader) error
	GetCover(ctx context.Context, coverID string) ([]byte, error)
}
