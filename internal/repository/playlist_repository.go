package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlaylistRepository interface {
	CreatePlaylist(ctx context.Context, params CreatePlaylistParams) (Playlist, error)
	GetPlaylist(ctx context.Context, playlistID int32) (Playlist, error)
	UpdatePlaylist(ctx context.Context, params UpdatePlaylistParams) (Playlist, error)
	DeletePlaylist(ctx context.Context, playlistID int32) error
	AddTrackToPlaylist(ctx context.Context, playlistID, trackID int32) error
	GetPlaylistTrackIDs(ctx context.Context, playlistID int32) ([]int32, error)
	GetSharedUsers(ctx context.Context, playlistID int32) ([]int32, error)
	GetUserOwnedPlaylists(ctx context.Context, ownerID int32) ([]PlaylistSummary, error)
	GetPublicPlaylists(ctx context.Context, userID int32) ([]PlaylistSummary, error)
	GetSharedPlaylists(ctx context.Context, userID int32) ([]PlaylistSummary, error)
	UserCanReadPlaylist(ctx context.Context, userID, playlistID int32) (bool, error)
	UserCanWritePlaylist(ctx context.Context, userID, playlistID int32) (bool, error)
	SharePlaylistWithMany(
		ctx context.Context, playlistID int32,
		hasWritePermission bool, users []int32,
	) error
	SharePlaylist(
		ctx context.Context, playlistID int32,
		hasWritePermission bool, userID int32,
	) error
	RevokePlaylistAccess(ctx context.Context, playlistID, userID int32) error
}

type CreatePlaylistParams struct {
	OwnerID  int32
	Name     string
	IsPublic bool
}

type UpdatePlaylistParams struct {
	ID   int32
	Name string
}

type Playlist struct {
	ID       int32
	Name     string
	OwnerID  int32
	IsPublic bool
}

type PlaylistSummary struct {
	ID      int32
	Name    string
	OwnerID int32
}

type PostgresPlaylistRepository struct {
	pool *pgxpool.Pool
	q    *db.Queries
}

func NewPlaylistRepository(pool *pgxpool.Pool) *PostgresPlaylistRepository {
	return &PostgresPlaylistRepository{
		pool: pool,
		q:    db.New(pool),
	}
}

func (repo *PostgresPlaylistRepository) CreatePlaylist(
	ctx context.Context, params CreatePlaylistParams,
) (Playlist, error) {
	playlist, err := repo.q.CreatePlaylist(ctx, db.CreatePlaylistParams{
		Name:     params.Name,
		OwnerID:  params.OwnerID,
		IsPublic: params.IsPublic,
	})
	if err != nil {
		return Playlist{}, wrapDBError(err)
	}
	return playlistFromDBPlaylist(playlist), nil
}

func (repo *PostgresPlaylistRepository) GetPlaylist(
	ctx context.Context, playlistID int32,
) (Playlist, error) {
	playlist, err := repo.q.GetPlaylist(ctx, playlistID)
	if err != nil {
		return Playlist{}, wrapDBError(err)
	}
	return Playlist{
		ID:      playlist.ID,
		Name:    playlist.Name,
		OwnerID: playlist.OwnerID,
	}, nil
}

func (repo *PostgresPlaylistRepository) UpdatePlaylist(
	ctx context.Context, params UpdatePlaylistParams,
) (Playlist, error) {
	playlist, err := repo.q.UpdatePlaylist(ctx, db.UpdatePlaylistParams{
		ID:   params.ID,
		Name: params.Name,
	})
	if err != nil {
		return Playlist{}, wrapDBError(err)
	}
	return playlistFromDBPlaylist(playlist), nil
}

func (repo *PostgresPlaylistRepository) DeletePlaylist(
	ctx context.Context, playlistID int32,
) error {
	err := repo.q.DeletePlaylist(ctx, playlistID)
	return wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) AddTrackToPlaylist(
	ctx context.Context, playlistID, trackID int32,
) error {
	err := repo.q.AddTrackToPlaylist(ctx, db.AddTrackToPlaylistParams{
		TrackID:    trackID,
		PlaylistID: playlistID,
	})
	return wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) GetPlaylistTrackIDs(
	ctx context.Context, playlistID int32,
) ([]int32, error) {
	tracks, err := repo.q.GetPlaylistWithTracks(ctx, playlistID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	trackIDs := make([]int32, len(tracks))
	for i, track := range tracks {
		trackIDs[i] = track.TrackID
	}
	return trackIDs, nil
}

func (repo *PostgresPlaylistRepository) GetSharedUsers(
	ctx context.Context, playlistID int32,
) ([]int32, error) {
	users, err := repo.q.GetSharedUsers(ctx, playlistID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	return users, nil
}

func (repo *PostgresPlaylistRepository) GetUserOwnedPlaylists(
	ctx context.Context, ownerID int32,
) ([]PlaylistSummary, error) {
	playlists, err := repo.q.GetUserOwnedPlaylists(ctx, ownerID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	summaries := make([]PlaylistSummary, len(playlists))
	for i, playlist := range playlists {
		summaries[i] = PlaylistSummary{
			ID:      playlist.ID,
			Name:    playlist.Name,
			OwnerID: ownerID,
		}
	}
	return summaries, nil
}

func (repo *PostgresPlaylistRepository) GetPublicPlaylists(
	ctx context.Context, userID int32,
) ([]PlaylistSummary, error) {
	playlists, err := repo.q.GetPublicPlaylists(ctx, userID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	summaries := make([]PlaylistSummary, len(playlists))
	for i, playlist := range playlists {
		summaries[i] = PlaylistSummary{
			ID:      playlist.ID,
			Name:    playlist.Name,
			OwnerID: playlist.OwnerID,
		}
	}
	return summaries, nil
}

func (repo *PostgresPlaylistRepository) GetSharedPlaylists(
	ctx context.Context, userID int32,
) ([]PlaylistSummary, error) {
	playlists, err := repo.q.GetSharedPlaylists(ctx, userID)
	if err != nil {
		return nil, wrapDBError(err)
	}
	summaries := make([]PlaylistSummary, len(playlists))
	for i, playlist := range playlists {
		summaries[i] = PlaylistSummary{
			ID:      playlist.ID,
			Name:    playlist.Name,
			OwnerID: playlist.OwnerID,
		}
	}
	return summaries, nil
}

func (repo *PostgresPlaylistRepository) UserCanReadPlaylist(
	ctx context.Context, userID, playlistID int32,
) (bool, error) {
	canRead, err := repo.q.UserCanReadPlaylist(ctx, db.UserCanReadPlaylistParams{
		UserID:     userID,
		PlaylistID: playlistID,
	})
	return canRead, wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) UserCanWritePlaylist(
	ctx context.Context, userID, playlistID int32,
) (bool, error) {
	canWrite, err := repo.q.UserCanWritePlaylist(ctx, db.UserCanWritePlaylistParams{
		UserID:     userID,
		PlaylistID: playlistID,
	})
	return canWrite, wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) SharePlaylistWithMany(
	ctx context.Context, playlistID int32,
	hasWritePermission bool, users []int32,
) error {
	_, err := withTx(ctx, repo.pool, repo.q, func(q *db.Queries) (int, error) {
		for _, userID := range users {
			err := q.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
				SharedWithUser:     userID,
				PlaylistID:         playlistID,
				HasWritePermission: hasWritePermission,
			})
			if err != nil {
				return 0, err
			}
		}
		return 0, nil
	})
	return wrapDBError(err)
}

func playlistFromDBPlaylist(playlist db.Playlist) Playlist {
	return Playlist{
		ID:       playlist.ID,
		Name:     playlist.Name,
		OwnerID:  playlist.OwnerID,
		IsPublic: playlist.IsPublic,
	}
}

func (repo *PostgresPlaylistRepository) SharePlaylist(
	ctx context.Context, playlistID int32,
	hasWritePermission bool, userID int32,
) error {
	err := repo.q.SharePlaylistWith(ctx, db.SharePlaylistWithParams{
		SharedWithUser:     userID,
		PlaylistID:         playlistID,
		HasWritePermission: hasWritePermission,
	})
	return wrapDBError(err)
}

func (repo *PostgresPlaylistRepository) RevokePlaylistAccess(
	ctx context.Context, playlistID, userID int32,
) error {
	_, err := repo.q.RevokePlaylistAccess(ctx, db.RevokePlaylistAccessParams{
		PlaylistID:     playlistID,
		SharedWithUser: userID,
	})
	return wrapDBError(err)
}
