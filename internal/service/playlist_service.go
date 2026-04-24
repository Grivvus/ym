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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type PlaylistCreateParams struct {
	OwnerID  int32
	Name     string
	IsPublic bool
}

type PlaylistService struct {
	queries        *db.Queries
	objStorage     storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewPlaylistService(q *db.Queries, st storage.Storage, logger *slog.Logger) PlaylistService {
	svc := PlaylistService{
		queries:    q,
		objStorage: st,
		logger:     logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *PlaylistService) loadArtworkOwner(
	ctx context.Context, id int32,
) (ArtworkOwner, error) {
	playlist, err := s.queries.GetPlaylist(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ArtworkOwner{}, NewErrNotFound("playlist", id)
		}
		return ArtworkOwner{}, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return ArtworkOwner{
		ID:   playlist.ID,
		Name: playlist.Name,
		Kind: "playlist",
	}, nil
}

func (s *PlaylistService) Create(
	ctx context.Context, playlistInfo PlaylistCreateParams,
	coverFileHeader *multipart.FileHeader,
) (api.PlaylistCreateResponse, error) {
	var ret api.PlaylistCreateResponse
	playlist, err := s.queries.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     playlistInfo.Name,
		OwnerID:  pgtype.Int4{Int32: playlistInfo.OwnerID, Valid: true},
		IsPublic: playlistInfo.IsPublic,
	})
	if err != nil {
		if e, ok := errors.AsType[*pgconn.PgError](err); ok && e.Code == "23505" {
			return ret, fmt.Errorf(
				"%w: user already has playlist with this name",
				NewErrAlreadyExists("playlist", playlistInfo.Name),
			)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret.PlaylistId = playlist.ID
	ret.CoverUploaded = coverFileHeader != nil

	if coverFileHeader == nil {
		return ret, nil
	}

	f, err := coverFileHeader.Open()
	if err != nil {
		return ret, fmt.Errorf("%w: assertion, must be nil", err)
	}
	defer func() { _ = f.Close() }()

	err = s.UploadCover(ctx, ret.PlaylistId, f)
	if err != nil {
		ret.CoverUploaded = false
	}

	return ret, nil
}

func (s *PlaylistService) AddTrack(ctx context.Context, playlistID, trackID int32) error {
	err := s.queries.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    trackID,
		PlaylistID: playlistID,
	})
	if err != nil {
		if e, ok := errors.AsType[*pgconn.PgError](err); ok && e.Code == "23505" {
			return fmt.Errorf(
				"%w: album already has a track",
				NewErrAlreadyExists("playlist", playlistID),
			)
		}
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (s *PlaylistService) Delete(
	ctx context.Context, playlistID int32,
) error {
	err := s.artworkService.Delete(ctx, playlistID)
	if err != nil {
		if _, ok := errors.AsType[ErrNotFound](err); !ok {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}
	err = s.queries.DeletePlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
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
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	ret.PlaylistId = playlist.ID
	ret.PlaylistName = playlist.Name
	ret.Tracks = []int32{}
	playlistTracks, err := s.queries.GetPlaylistWithTracks(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("playlist", playlistID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	for _, track := range playlistTracks {
		ret.Tracks = append(ret.Tracks, track.TrackID)
	}
	return ret, nil
}

func (s *PlaylistService) ChangePlaylist(
	ctx context.Context, userID, playlistID int32,
	newPlaylistData api.PlaylistUpdateRequest,
) (api.PlaylistResponse, error) {
	playlistRow, err := s.queries.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return api.PlaylistResponse{}, NewErrNotFound("playlist", playlistID)
		}
		return api.PlaylistResponse{}, fmt.Errorf("%w, exact - %w", ErrUnknownDBError, err)
	}
	// if owner is none - only superuser should be able to modify it
	if playlistRow.OwnerID.Int32 != userID {
		return api.PlaylistResponse{}, fmt.Errorf("%w, you can't modify this playlist", ErrUnauthorized)
	}
	updatedPlaylist, err := s.queries.UpdatePlaylist(ctx, db.UpdatePlaylistParams{
		ID:   playlistID,
		Name: newPlaylistData.PlaylistName,
	})
	if err != nil {
		return api.PlaylistResponse{}, fmt.Errorf("%w, exact - %w", ErrUnknownDBError, err)
	}
	return api.PlaylistResponse{
		PlaylistId:   updatedPlaylist.ID,
		PlaylistName: updatedPlaylist.Name,
	}, nil
}

func (s *PlaylistService) GetUserPlaylists(ctx context.Context, userID int32) (api.Playlists, error) {
	playlists, err := s.queries.GetUserPlaylists(ctx, pgtype.Int4{Int32: userID, Valid: true})
	if err != nil {
		return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
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
	return s.artworkService.Upload(ctx, playlistID, cover)
}

func (s *PlaylistService) DeleteCover(
	ctx context.Context, playlistID int32,
) error {
	return s.artworkService.Delete(ctx, playlistID)
}

func (s *PlaylistService) GetCover(
	ctx context.Context, playlistID int32,
) ([]byte, error) {
	return s.artworkService.Get(ctx, playlistID)
}
