package handlers

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/repository" // should move models and interfaces from repository to domain or something similar
	"github.com/Grivvus/ym/internal/service"
	"github.com/samber/lo"
)

type SearchHandler struct {
	logger  *slog.Logger
	service service.SearchService
}

func (h SearchHandler) Search(w http.ResponseWriter, r *http.Request, params api.SearchParams) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}

	searchParams := repository.SearchParams{
		UserID: userID,
		Query:  params.Q,
	}
	if params.Limit == nil {
		searchParams.LimitBound = 20 // default value
	} else {
		searchParams.LimitBound = *params.Limit
	}

	if params.Types == nil {
		// default all is on
		searchParams.IncludeAlbums = true
		searchParams.IncludeArtists = true
		searchParams.IncludeTracks = true
	} else {
		types := *params.Types
		if slices.Contains(types, "albums") {
			searchParams.IncludeAlbums = true
		}
		if slices.Contains(types, "artists") {
			searchParams.IncludeArtists = true
		}
		if slices.Contains(types, "tracks") {
			searchParams.IncludeTracks = true
		}
	}

	searchResult, err := h.service.Search(r.Context(), searchParams)
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}

	if err := WriteJSON(w, http.StatusOK, searchResponseFromRepositorySearchResult(searchResult)); err != nil {
		h.logger.Error("can't encode search response", "err", err)
	}
}

func searchResponseFromRepositorySearchResult(result repository.SearchResult) api.SearchResponse {
	return api.SearchResponse{
		Query: result.Query,
		Tracks: lo.Map(
			result.Tracks,
			func(track repository.TrackSearchResult, _ int) api.SearchTrackResult {
				return api.SearchTrackResult{
					TrackId:             track.TrackID,
					Name:                track.TrackName,
					AlbumId:             track.AlbumID,
					AlbumName:           track.AlbumName,
					ArtistId:            track.ArtistID,
					ArtistName:          track.ArtistName,
					DurationMs:          track.TrackDurationMS,
					IsGloballyAvailable: track.IsGloballyAvailable,
					Score:               track.SearchScore,
				}
			},
		),
		Albums: lo.Map(
			result.Albums,
			func(album repository.AlbumSearchResult, _ int) api.SearchAlbumResult {
				return api.SearchAlbumResult{
					AlbumId:     album.AlbumID,
					AlbumName:   album.AlbumName,
					ArtistId:    album.ArtistID,
					ArtistName:  album.ArtistName,
					ReleaseYear: album.ReleaseYear,
					Score:       album.SearchScore,
				}
			},
		),
		Artists: lo.Map(
			result.Artists,
			func(artist repository.ArtistSearchResult, _ int) api.SearchArtistResult {
				return api.SearchArtistResult{
					ArtistId:   artist.ArtistID,
					ArtistName: artist.ArtistName,
					Score:      artist.SearchScore,
				}
			},
		),
	}
}
