package storage

import (
	"github.com/minio/minio-go/v7"
)

type minioStorage struct {
	client *minio.Client
}
