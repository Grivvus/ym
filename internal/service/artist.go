package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/db"
	"github.com/Grivvus/ym/internal/storage"
	"github.com/jackc/pgx/v5"
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

func (s *ArtistService) Get(ctx context.Context, id int) (api.ArtistInfoResponse, error) {
	var ret api.ArtistInfoResponse

	var (
		artistID   int
		artistName string
	)

	artistWithAlbums, err := s.queries.GetArtistWithAlbums(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			artist, err := s.queries.GetArtist(ctx, int32(id))
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					return ret, NewErrNotFound("artist", id)
				}
				return ret, fmt.Errorf("unkown server error: %w", err)
			}
			artistID = int(artist.ID)
			artistName = artist.Name
		} else {
			return ret, fmt.Errorf("unkown server error: %w", err)
		}
	}

	albums := make([]int, len(artistWithAlbums))
	for i, album := range artistWithAlbums {
		albums[i] = int(album.AlbumID)
	}

	if len(artistWithAlbums) > 0 {
		artistID = int(artistWithAlbums[0].ArtistID)
		artistName = artistWithAlbums[0].ArtistName
	}

	ret.ArtistId = artistID
	ret.ArtistName = artistName

	artistImageID := getArtistImageID(artistID, artistName)
	if s.st.ImageExist(ctx, artistImageID) {
		url := fmt.Sprintf("http://my_url/?type=image,id=%v", artistImageID)
		ret.ArtistCoverUrl = &url
	} else {
		ret.ArtistCoverUrl = nil
	}

	ret.ArtistAlbums = albums

	return ret, nil
}

func (s *ArtistService) Delete(ctx context.Context, id int) (api.ArtistDeleteResponse, error) {
	ret := api.ArtistDeleteResponse{ArtistId: id}
	artist, err := s.queries.GetArtist(ctx, int32(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, NewErrNotFound("artist", id)
		}
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	err = s.st.RemoveImage(ctx, getArtistImageID(int(artist.ID), artist.Name))
	if err != nil {
		return ret, fmt.Errorf("can't delete artist image: %w", err)
	}
	err = s.queries.DeleteArtist(ctx, int32(id))
	if err != nil {
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	return ret, nil
}

func (s *ArtistService) Create(
	ctx context.Context, artistInfo api.ArtistCreateRequest,
) (api.ArtistCreateResponse, error) {
	var ret api.ArtistCreateResponse
	artist, err := s.queries.CreateArtist(ctx, artistInfo.ArtistName)
	if err != nil {
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	if artistInfo.ArtistImage != nil {
		// upload artist's image
	}
	ret.ArtistId = int(artist.ID)
	return ret, nil
}

func getArtistImageID(artistID int, artistName string) string {
	return fmt.Sprintf("%v_%v", artistID, artistName)
}
