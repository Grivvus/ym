package service

import (
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
)

type AlbumService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewAlbumService(q *db.Queries, st storage.Storage) AlbumService {
	return AlbumService{
		queries: q,
		st:      st,
	}
}
