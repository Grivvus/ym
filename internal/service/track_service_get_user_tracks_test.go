package service

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetUserTracksUsesAllTracksForSuperuser(t *testing.T) {
	t.Parallel()

	trackRepo := &trackRepositorySpy{
		allTracks: []repository.Track{
			{ID: 1, Name: "visible", ArtistID: 10},
			{ID: 2, Name: "private", ArtistID: 20},
		},
		albumIDs: map[int32]int32{1: 100, 2: 200},
	}
	service := TrackService{
		repo:     trackRepo,
		userRepo: userRepositoryStub{user: repository.User{ID: 7, IsSuperuser: true}},
		logger:   slog.Default(),
	}

	got, err := service.GetUserTracks(context.Background(), 7)

	require.NoError(t, err)
	assert.True(t, trackRepo.allTracksCalled)
	assert.False(t, trackRepo.userTracksCalled)
	assert.Equal(t, []api.TrackMetadata{
		{TrackId: 1, Name: "visible", ArtistId: 10, AlbumId: 100},
		{TrackId: 2, Name: "private", ArtistId: 20, AlbumId: 200},
	}, got)
}

func TestGetUserTracksUsesAccessibleTracksForRegularUser(t *testing.T) {
	t.Parallel()

	trackRepo := &trackRepositorySpy{
		userTracks: []repository.Track{
			{ID: 1, Name: "visible", ArtistID: 10},
		},
		albumIDs: map[int32]int32{1: 100},
	}
	service := TrackService{
		repo:     trackRepo,
		userRepo: userRepositoryStub{user: repository.User{ID: 7}},
		logger:   slog.Default(),
	}

	got, err := service.GetUserTracks(context.Background(), 7)

	require.NoError(t, err)
	assert.False(t, trackRepo.allTracksCalled)
	assert.True(t, trackRepo.userTracksCalled)
	assert.Equal(t, []api.TrackMetadata{
		{TrackId: 1, Name: "visible", ArtistId: 10, AlbumId: 100},
	}, got)
}

type trackRepositorySpy struct {
	userTracksCalled bool
	allTracksCalled  bool
	userTracks       []repository.Track
	allTracks        []repository.Track
	albumIDs         map[int32]int32
}

func (r *trackRepositorySpy) CreateTrackWithAlbum(
	context.Context, repository.CreateTrackParams,
) (repository.Track, error) {
	panic("unexpected call")
}

func (r *trackRepositorySpy) AddToTranscodingQueue(context.Context, int32, string) error {
	panic("unexpected call")
}

func (r *trackRepositorySpy) GetTrack(context.Context, int32) (repository.Track, error) {
	panic("unexpected call")
}

func (r *trackRepositorySpy) GetUserTracks(
	context.Context, int32,
) ([]repository.Track, error) {
	r.userTracksCalled = true
	return r.userTracks, nil
}

func (r *trackRepositorySpy) CanUserAccessTrack(context.Context, int32, int32) (bool, error) {
	panic("unexpected call")
}

func (r *trackRepositorySpy) GetAllTracks(context.Context) ([]repository.Track, error) {
	r.allTracksCalled = true
	return r.allTracks, nil
}

func (r *trackRepositorySpy) GetAlbumIDByTrackID(
	_ context.Context, trackID int32,
) (int32, error) {
	return r.albumIDs[trackID], nil
}

func (r *trackRepositorySpy) DeleteTrack(context.Context, int32) error {
	panic("unexpected call")
}

type userRepositoryStub struct {
	user repository.User
}

func (r userRepositoryStub) GetAllUsers(context.Context) ([]repository.UserSummary, error) {
	panic("unexpected call")
}

func (r userRepositoryStub) GetUserByID(context.Context, int32) (repository.User, error) {
	return r.user, nil
}

func (r userRepositoryStub) UpdateUser(
	context.Context, repository.UpdateUserParams,
) (repository.User, error) {
	panic("unexpected call")
}

func (r userRepositoryStub) UpdateUserPassword(
	context.Context, repository.UpdateUserPasswordParams,
) error {
	panic("unexpected call")
}
