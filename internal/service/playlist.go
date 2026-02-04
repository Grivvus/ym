package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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

func (s *PlaylistService) Create(
	ctx context.Context, playlistInfo api.PlaylistCreateRequest,
) (api.PlaylistCreateResponse, error) {
	playlist, err := s.queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:    playlistInfo.PlaylistName,
		OwnerID: pgtype.Int4{Int32: int32(playlistInfo.OwnerId), Valid: true},
	})
	if err != nil {
		return api.PlaylistCreateResponse{}, fmt.Errorf("can't create playlist: %w", err)
	}
	return api.PlaylistCreateResponse{PlaylistId: int(playlist.ID)}, nil
}

func (s *PlaylistService) Delete(
	ctx context.Context, playlistID int,
) error {
	err := s.queries.DeletePlaylist(ctx, int32(playlistID))
	if err != nil {
		return fmt.Errorf("can't delete playlist: %w", err)
	}
	return nil
}

func (s *PlaylistService) Get(
	ctx context.Context, playlistID int,
) (api.PlaylistInfoResponse, error) {
	var ret api.PlaylistInfoResponse
	playlistTracks, err := s.queries.GetPlaylistWithTracks(ctx, int32(playlistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("playlist", playlistID)
		} else {
			return ret, fmt.Errorf("unkown server error: %w", err)
		}
	}

	ret.PlaylistId = int(playlistTracks[0].ID)
	ret.PlaylistName = playlistTracks[0].Name
	for _, track := range playlistTracks {
		ret.Tracks = append(ret.Tracks, int(track.TrackID))
	}
	return ret, nil
}

func (s *PlaylistService) UploadCover(
	ctx context.Context, playlistID int, cover io.Reader,
) error {
	playlist, err := s.queries.GetPlaylist(ctx, int32(playlistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		} else {
			return fmt.Errorf("can't upload image, cause: %w", err)
		}
	}
	rcTranscoded, err := transcoder.FromBase64(cover)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	defer func() { _ = rcTranscoded.Close() }()

	err = s.st.PutImage(ctx, ImageID("playlist", int(playlist.ID), playlist.Name), rcTranscoded)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	return nil
}

func (s *PlaylistService) DeleteCover(
	ctx context.Context, playlistID int,
) error {
	playlist, err := s.queries.GetPlaylist(ctx, int32(playlistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	err = s.st.RemoveImage(ctx, ImageID("playlist", playlistID, playlist.Name))
	if err != nil {
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	return nil
}

func (s *PlaylistService) GetCover(
	ctx context.Context, playlistID int,
) ([]byte, error) {
	playlist, err := s.queries.GetPlaylist(ctx, int32(playlistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NewErrNotFound("playlist", playlistID)
		}
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	bimage, err := s.st.GetImage(ctx, ImageID("album", int(playlist.ID), playlist.Name))
	if err != nil {
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	return bimage, nil
}
