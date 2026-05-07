package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/Grivvus/ym/internal/storage"
)

type PlaylistCreateParams struct {
	OwnerID  int32
	Name     string
	IsPublic bool
}

type PlaylistService struct {
	repo           repository.PlaylistRepository
	objStorage     storage.Storage
	logger         *slog.Logger
	artworkService ArtworkManager
}

func NewPlaylistService(
	playlistRepository repository.PlaylistRepository, st storage.Storage,
	logger *slog.Logger,
) PlaylistService {
	svc := PlaylistService{
		repo:       playlistRepository,
		objStorage: st,
		logger:     logger,
	}

	svc.artworkService = NewArtworkManager(st, svc.loadArtworkOwner, logger)

	return svc
}

func (s *PlaylistService) loadArtworkOwner(
	ctx context.Context, id int32,
) (ArtworkOwner, error) {
	playlist, err := s.repo.GetPlaylist(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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
	playlist, err := s.repo.CreatePlaylist(ctx, repository.CreatePlaylistParams{
		Name:     playlistInfo.Name,
		OwnerID:  playlistInfo.OwnerID,
		IsPublic: playlistInfo.IsPublic,
	})
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
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

	err = s.UploadCover(ctx, ret.PlaylistId, playlist.OwnerID, f)
	if err != nil {
		ret.CoverUploaded = false
	}

	return ret, nil
}

func (s *PlaylistService) AddTrack(ctx context.Context, playlistID, userID, trackID int32) error {
	err := s.checkUserHasWritePermissions(ctx, playlistID, userID)
	if err != nil {
		return err
	}
	err = s.repo.AddTrackToPlaylist(ctx, playlistID, trackID)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
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
	ctx context.Context, playlistID, userID int32,
) error {
	err := s.requireToBeOwner(ctx, playlistID, userID)
	if err != nil {
		return err
	}
	err = s.artworkService.Delete(ctx, playlistID)
	if err != nil {
		if _, ok := errors.AsType[ErrNotFound](err); !ok {
			return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}
	err = s.repo.DeletePlaylist(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	return nil
}

func (s *PlaylistService) Get(
	ctx context.Context, playlistID int32,
) (api.PlaylistWithTracksResponse, error) {
	var ret api.PlaylistWithTracksResponse
	playlist, err := s.repo.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ret, NewErrNotFound("playlist", playlistID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	usersSharedWith, err := s.repo.GetSharedUsers(ctx, playlistID)
	if err != nil {
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}
	if usersSharedWith == nil {
		usersSharedWith = make([]int32, 0)
	}
	ret.PlaylistId = playlist.ID
	ret.PlaylistName = playlist.Name
	ret.SharedWith = usersSharedWith
	ret.Tracks = []int32{}
	playlistTracks, err := s.repo.GetPlaylistTrackIDs(ctx, playlistID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ret, NewErrNotFound("playlist", playlistID)
		}
		return ret, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
	}

	for _, track := range playlistTracks {
		ret.Tracks = append(ret.Tracks, track)
	}
	return ret, nil
}

func (s *PlaylistService) ChangePlaylist(
	ctx context.Context, userID, playlistID int32,
	newPlaylistData api.PlaylistUpdateRequest,
) (api.PlaylistResponse, error) {
	err := s.checkUserHasWritePermissions(ctx, playlistID, userID)
	if err != nil {
		return api.PlaylistResponse{}, err
	}
	updatedPlaylist, err := s.repo.UpdatePlaylist(ctx, repository.UpdatePlaylistParams{
		ID:   playlistID,
		Name: newPlaylistData.PlaylistName,
	})
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return api.PlaylistResponse{}, NewErrNotFound("playlist", playlistID)
		}
		if errors.Is(err, repository.ErrAlreadyExists) {
			return api.PlaylistResponse{}, NewErrAlreadyExists("playlist", newPlaylistData.PlaylistName)
		}
		return api.PlaylistResponse{}, fmt.Errorf("%w, exact - %w", ErrUnknownDBError, err)
	}
	return api.PlaylistResponse{
		PlaylistId:   updatedPlaylist.ID,
		PlaylistName: updatedPlaylist.Name,
	}, nil
}

func (s *PlaylistService) GetUserPlaylists(
	ctx context.Context, userID int32, params api.GetPlaylistsParams,
) (api.Playlists, error) {
	var owned []repository.PlaylistSummary
	var global []repository.PlaylistSummary
	var shared []repository.PlaylistSummary
	var err error
	if params.IncludeOwned != nil && *params.IncludeOwned {
		owned, err = s.repo.GetUserOwnedPlaylists(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}
	if params.IncludePublic != nil && *params.IncludePublic {
		global, err = s.repo.GetPublicPlaylists(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}
	if params.IncludeShared != nil && *params.IncludeShared {
		shared, err = s.repo.GetSharedPlaylists(ctx, userID)
		if err != nil {
			return nil, fmt.Errorf("%w caused by: %w", ErrUnknownDBError, err)
		}
	}
	unique := make(map[int32]struct{}, len(shared))
	ret := make(api.Playlists, 0, len(owned)+len(global)+len(shared))
	for _, playlist := range owned {
		ret = append(ret, api.ExtendedPlaylist{
			PlaylistId:      playlist.ID,
			PlaylistName:    playlist.Name,
			PlaylistOwnerId: playlist.OwnerID,
			PlaylistType:    api.Owned,
		})
	}
	for _, playlist := range shared {
		unique[playlist.ID] = struct{}{}
		ret = append(ret, api.ExtendedPlaylist{
			PlaylistId:      playlist.ID,
			PlaylistName:    playlist.Name,
			PlaylistOwnerId: playlist.OwnerID,
			PlaylistType:    api.Shared,
		})
	}
	for _, playlist := range global {
		if _, ok := unique[playlist.ID]; ok {
			continue
		}
		ret = append(ret, api.ExtendedPlaylist{
			PlaylistId:      playlist.ID,
			PlaylistName:    playlist.Name,
			PlaylistOwnerId: playlist.OwnerID,
			PlaylistType:    api.Public,
		})
	}
	return ret, nil
}

func (s *PlaylistService) SharePlaylistWithUsers(
	ctx context.Context, playlistID int32, ownerID int32,
	shareInfo api.PlaylistShareRequest,
) error {
	err := s.requireToBeOwner(ctx, playlistID, ownerID)
	if err != nil {
		return err
	}
	err = s.repo.SharePlaylistWithMany(
		ctx, playlistID, shareInfo.HasWritePermission, shareInfo.ShareWithUsers,
	)
	if err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return fmt.Errorf(
				"%w - playlist already shared with some users you chose",
				NewErrAlreadyExists("playlist-shared_user", playlistID),
			)
		}
		return fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}

	return nil
}

func (s *PlaylistService) RevokePlaylistAccess(
	ctx context.Context, playlistID, ownerID, userID int32,
) error {
	err := s.requireToBeOwner(ctx, playlistID, ownerID)
	if err != nil {
		return err
	}
	err = s.repo.RevokePlaylistAccess(ctx, playlistID, userID)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}
	return nil
}

func (s *PlaylistService) UploadCover(
	ctx context.Context, userID, playlistID int32, cover io.Reader,
) error {
	err := s.checkUserHasWritePermissions(ctx, playlistID, userID)
	if err != nil {
		return err
	}
	return s.artworkService.Upload(ctx, playlistID, cover)
}

func (s *PlaylistService) DeleteCover(
	ctx context.Context, userID, playlistID int32,
) error {
	err := s.checkUserHasWritePermissions(ctx, playlistID, userID)
	if err != nil {
		return err
	}
	return s.artworkService.Delete(ctx, playlistID)
}

func (s *PlaylistService) GetCover(
	ctx context.Context, playlistID int32,
) ([]byte, error) {
	return s.artworkService.Get(ctx, playlistID)
}

func (s *PlaylistService) requireToBeOwner(ctx context.Context, playlistID, userID int32) error {
	playlist, err := s.repo.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewErrNotFound("playlist", playlistID)
		}
		return fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}
	if playlist.OwnerID != userID {
		return fmt.Errorf("%w: you can't manage someone else's playlist", ErrUnauthorized)
	}
	return nil
}

func (s *PlaylistService) checkUserHasWritePermissions(
	ctx context.Context, playlistID, userID int32,
) error {
	playlist, err := s.repo.GetPlaylist(ctx, playlistID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return NewErrNotFound("playlist", playlistID)
		}
		return fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}
	if playlist.OwnerID == userID {
		return nil
	}
	sharedWith, err := s.repo.GetSharedUsers(ctx, playlistID)
	if err != nil {
		return fmt.Errorf("%w, caused by - %w", ErrUnknownDBError, err)
	}
	for _, id := range sharedWith {
		if id == userID {
			return nil
		}
	}
	return fmt.Errorf("%w: you can't manage someone else's playlist", ErrUnauthorized)
}
