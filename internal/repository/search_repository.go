package repository

import (
	"context"

	"github.com/Grivvus/ym/internal/db"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"
)

type SearchRepository interface {
	Search(context.Context, SearchParams) (SearchResult, error)
}

type SearchParams struct {
	UserID         int32
	Query          string
	LimitBound     int
	IncludeTracks  bool
	IncludeAlbums  bool
	IncludeArtists bool
}

type SearchResult struct {
	Query   string
	Tracks  []TrackSearchResult
	Albums  []AlbumSearchResult
	Artists []ArtistSearchResult
}

type TrackSearchResult struct {
	TrackID             int32
	TrackName           string
	AlbumID             int32
	AlbumName           string
	ArtistID            int32
	ArtistName          string
	TrackDurationMS     int64
	IsGloballyAvailable bool
	SearchScore         float32
}

type AlbumSearchResult struct {
	AlbumID     int32
	AlbumName   string
	ArtistID    int32
	ArtistName  string
	ReleaseYear int
	SearchScore float32
}

type ArtistSearchResult struct {
	ArtistID    int32
	ArtistName  string
	SearchScore float32
}

type PostgresSearchRepository struct {
	pool    *pgxpool.Pool
	queries *db.Queries
}

func NewSearchRepository(pool *pgxpool.Pool) SearchRepository {
	return &PostgresSearchRepository{
		pool:    pool,
		queries: db.New(pool),
	}
}

func (s *PostgresSearchRepository) Search(
	ctx context.Context, params SearchParams,
) (SearchResult, error) {
	tracksFound := make([]TrackSearchResult, 0)
	albumsFound := make([]AlbumSearchResult, 0)
	artistsFound := make([]ArtistSearchResult, 0)
	if params.IncludeTracks {
		tracks, err := s.queries.SearchTracks(ctx, db.SearchTracksParams{
			UserID: params.UserID,
			Query:  params.Query,
			Bound:  int32(params.LimitBound),
		})
		if err != nil {
			return SearchResult{}, wrapDBError(err)
		}
		tracksFound = lo.Map(tracks, func(track db.SearchTracksRow, i int) TrackSearchResult {
			return TrackSearchResult{
				TrackID:             track.ID,
				TrackName:           track.Name,
				ArtistID:            track.ArtistID,
				ArtistName:          track.ArtistName,
				AlbumID:             track.AlbumID,
				AlbumName:           track.AlbumName,
				IsGloballyAvailable: track.IsGloballyAvailable,
				TrackDurationMS:     int64(track.DurationMs.Int32),
				SearchScore:         track.Score,
			}
		})
	}

	if params.IncludeArtists {
		artists, err := s.queries.SearchArtists(ctx, db.SearchArtistsParams{
			Query: params.Query,
			Bound: int32(params.LimitBound),
		})
		if err != nil {
			return SearchResult{}, wrapDBError(err)
		}
		artistsFound = lo.Map(artists, func(artist db.SearchArtistsRow, i int) ArtistSearchResult {
			return ArtistSearchResult{
				ArtistID:    artist.ArtistID,
				ArtistName:  artist.ArtistName,
				SearchScore: artist.Score,
			}
		})
	}

	if params.IncludeAlbums {
		albums, err := s.queries.SearchAlbums(ctx, db.SearchAlbumsParams{
			Query: params.Query,
			Bound: int32(params.LimitBound),
		})
		if err != nil {
			return SearchResult{}, wrapDBError(err)
		}
		albumsFound = lo.Map(albums, func(album db.SearchAlbumsRow, i int) AlbumSearchResult {
			return AlbumSearchResult{
				AlbumID:     album.AlbumID,
				AlbumName:   album.Name,
				ArtistID:    album.ArtistID,
				ArtistName:  album.ArtistName,
				ReleaseYear: int(album.ReleaseYear.Int32),
				SearchScore: album.Score,
			}
		})
	}

	return SearchResult{
		Query:   params.Query,
		Tracks:  tracksFound,
		Albums:  albumsFound,
		Artists: artistsFound,
	}, nil
}
