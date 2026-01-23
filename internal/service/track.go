package service

import (
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
)

type TrackService struct {
	queries *db.Queries
	st      storage.Storage
}

func NewTrackService(q *db.Queries, st storage.Storage) TrackService {
	return TrackService{
		queries: q,
		st:      st,
	}
}
