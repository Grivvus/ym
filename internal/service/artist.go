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

	artistImageID := ImageID("artist", artistID, artistName)
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
		// delete artist's, that doesn't exist is noop
		if errors.Is(err, pgx.ErrNoRows) {
			return ret, nil
		}
		return ret, fmt.Errorf("unkown server error: %w", err)
	}
	/* could i make this 2 ops transactional? */
	err = s.st.RemoveImage(ctx, ImageID("artist", int(artist.ID), artist.Name))
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

	ret.ArtistId = int(artist.ID)

	if artistInfo.ArtistImage != nil {
		rc, err := artistInfo.ArtistImage.Reader()
		if err != nil {
			// assertion
			panic(err)
		}
		defer func() { _ = rc.Close() }()

		err = s.UploadImage(ctx, ret.ArtistId, rc)
		if err != nil {
			return ret, err
		}
	}
	return ret, nil
}

func (s *ArtistService) UploadImage(
	ctx context.Context, artistID int, file io.Reader,
) error {
	artist, err := s.queries.GetArtist(ctx, int32(artistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		} else {
			return fmt.Errorf("can't upload image, cause: %w", err)
		}
	}
	rcTranscoded, err := transcoder.FromBase64(file)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	defer func() { _ = rcTranscoded.Close() }()

	err = s.st.PutImage(ctx, ImageID("artist", int(artist.ID), artist.Name), rcTranscoded)
	if err != nil {
		return fmt.Errorf("can't upload image, cause: %w", err)
	}
	return nil
}

func (s *ArtistService) DeleteImage(
	ctx context.Context, artistID int,
) error {
	artist, err := s.queries.GetArtist(ctx, int32(artistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	err = s.st.RemoveImage(ctx, ImageID("artist", artistID, artist.Name))
	if err != nil {
		return fmt.Errorf("can't delete image, cause: %w", err)
	}
	return nil
}

func (s *ArtistService) GetImage(
	ctx context.Context, artistID int,
) ([]byte, error) {
	artist, err := s.queries.GetArtist(ctx, int32(artistID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, NewErrNotFound("artist", artistID)
		}
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	bimage, err := s.st.GetImage(ctx, ImageID("artist", int(artist.ID), artist.Name))
	if err != nil {
		return nil, fmt.Errorf("unkown server error: %w", err)
	}
	return bimage, nil
}
