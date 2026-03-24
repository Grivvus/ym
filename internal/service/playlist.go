package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/Grivvus/ym/internal/transcoder"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PlaylistCreateParams struct {
	OwnerID int32
	Name    string
}

type PlaylistService struct {
	queries *db.Queries
	st      storage.Storage
	logger  *slog.Logger
}

func NewPlaylistService(q *db.Queries, st storage.Storage, logger *slog.Logger) PlaylistService {
	return PlaylistService{
		queries: q,
		st:      st,
		logger:  logger,
	}
}

func (s *PlaylistService) Create(
	ctx context.Context, playlistInfo PlaylistCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.PlaylistCreateResponse, error) {
	var ret api.PlaylistCreateResponse
	playlist, err := s.queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:    playlistInfo.Name,
		OwnerID: pgtype.Int4{Int32: playlistInfo.OwnerID, Valid: true},
	})
	if err != nil {
		return ret, fmt.Errorf("can't create playlist: %w", err)
	}
	ret.PlaylistId = playlist.ID
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

func (s *PlaylistService) AddTrack(ctx context.Context, playlistID, trackID int32) error {
	err := s.queries.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    trackID,
		PlaylistID: playlistID,
	})
	return err
}

func (s *PlaylistService) Delete(
	ctx context.Context, playlistID int32,
) error {
	err := s.queries.DeletePlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("can't delete playlist: %w", err)
	}
	return nil
}

func (s *PlaylistService) Get(
	ctx context.Context, playlistID int32,
) (api.PlaylistWithTracksResponse, error) {
	var ret api.PlaylistWithTracksResponse
	playlist, err := s.queries.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("playlist", playlistID)
		}
		return ret, fmt.Errorf("unknown server error: %w", err)
	}
	ret.PlaylistId = playlist.ID
	ret.PlaylistName = playlist.Name
	ret.Tracks = []int32{}
	playlistTracks, err := s.queries.GetPlaylistWithTracks(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("playlist", playlistID)
		}
		return ret, fmt.Errorf("unknown server error: %w", err)
	}

	for _, track := range playlistTracks {
		ret.Tracks = append(ret.Tracks, track.TrackID)
	}
	return ret, nil
}

func (s *PlaylistService) GetUserPlaylists(ctx context.Context, userID int32) (api.Playlists, error) {
	playlists, err := s.queries.GetUserPlaylists(ctx, pgtype.Int4{Int32: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("unknown server error: %w", err)
	}
	ret := make(api.Playlists, len(playlists))
	for i, playlist := range playlists {
		ret[i] = api.PlaylistResponse{
			PlaylistId:   playlist.ID,
			PlaylistName: playlist.Name,
		}
	}
	return ret, nil
}

func (s *PlaylistService) UploadCover(
	ctx context.Context, playlistID int32, cover io.Reader,
) error {
	playlist, err := s.queries.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("can't upload image, cause: %w", err)
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
	ctx context.Context, playlistID int32,
) error {
	playlist, err := s.queries.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	err = s.st.RemoveImage(ctx, ImageID("playlist", int(playlistID), playlist.Name))
	if err != nil {
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	return nil
}

func (s *PlaylistService) GetCover(
	ctx context.Context, playlistID int32,
) ([]byte, error) {
	playlist, err := s.queries.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NewErrNotFound("playlist", playlistID)
		}
		return nil, fmt.Errorf("unknown server error: %w", err)
	}
	bimage, err := s.st.GetImage(ctx, ImageID("playlist", int(playlist.ID), playlist.Name))
	if err != nil {
		return nil, fmt.Errorf("unknown server error: %w", err)
	}
	return bimage, nil
}
