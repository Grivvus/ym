package service

import (
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
)

type PlaylistService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewPlaylistService(q *db.Queries, st storage.Storage) PlaylistService {
	return PlaylistService{
		queries: q,
		st:      st,
	}
}
