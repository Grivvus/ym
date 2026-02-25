package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PlaylistCreateParams struct {
	OwnerID int
	Name    string
}

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
	ctx context.Context, playlistInfo PlaylistCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.PlaylistCreateResponse, error) {
	var ret api.PlaylistCreateResponse
	playlist, err := s.queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:    playlistInfo.Name,
		OwnerID: pgtype.Int4{Int32: int32(playlistInfo.OwnerID), Valid: true},
	})
	if err != nil {
		return ret, fmt.Errorf("can't create playlist: %w", err)
	}
	ret.PlaylistId = int(playlist.ID)
	// no cover was provided
	if coverFileHeader == nil {
		return ret, nil
	}

	f, err := coverFileHeader.Open()
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()

	err = s.UploadCover(ctx, ret.PlaylistId, f)
	if err != nil {
		go func() {
			_ = s.DeleteCover(ctx, ret.PlaylistId)
		}()
		return ret, err
	}

	return ret, nil
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
	rcTranscoded, err := transcoder.ToWebp(cover)
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
	bimage, err := s.st.GetImage(ctx, ImageID("playlist", int(playlist.ID), playlist.Name))
	if err != nil {
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	return bimage, nil
}
