package service

import (
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
)

type ArtistService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewArtistService(q *db.Queries, st storage.Storage) ArtistService {
	return ArtistService{
		queries: q,
		st:      st,
	}
}
