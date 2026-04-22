package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

const defaultArtistLimit = 5

type ArtistHandlers struct {
	artistService service.ArtistService
	logger        *slog.Logger
}

func (h ArtistHandlers) CreateArtist(w http.ResponseWriter, r *http.Request) {
	// if there's no artist_image it's still ok
	_, fileHeader, _ := r.FormFile("artist_image")
	artistName := r.FormValue("artist_name")
	if artistName == "" {
		_ = WriteError(w, http.StatusBadRequest, errors.New("artist name is required"))
		return
	}

	artistResponse, err := h.artistService.Create(r.Context(), artistName, fileHeader)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = WriteError(w, http.StatusConflict, err)
			return
		}
		_ = WriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create new artist: %w", err),
		)
		return
	}

	err = WriteJSON(w, http.StatusCreated, artistResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetAllArtists(
	w http.ResponseWriter, r *http.Request, params api.GetAllArtistsParams,
) {
	var artists []api.ArtistInfoResponse
	var err error
	if params.StartsWith == nil {
		h.logger.Debug("general")
		artists, err = h.artistService.GetAll(r.Context())
	} else {
		limit := defaultArtistLimit
		if params.Limit != nil {
			limit = *params.Limit
		}
		h.logger.Debug("with filters")
		artists, err = h.artistService.GetWithFilters(r.Context(), *params.StartsWith, limit)
	}
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
	}
	err = WriteJSON(w, http.StatusOK, artists)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	response, err := h.artistService.Delete(r.Context(), artistID)
	if err != nil {
		_ = WriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete artist: %w", err),
		)
		return
	}
	err = WriteJSON(w, http.StatusOK, response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	response, err := h.artistService.Get(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find artist with this id: %w", err),
			)
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}
	err = WriteJSON(w, http.StatusOK, response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	err := h.artistService.DeleteImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find artist with this id: %w", err),
			)
		} else {
			_ = WriteError(
				w, http.StatusInternalServerError,
				fmt.Errorf("can't delete artist image: %w", err),
			)
		}
		return
	}
	err = WriteJSON(w, http.StatusOK, api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	err := h.artistService.UploadImage(r.Context(), artistID, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, err)
			return
		}
		_ = WriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't upload artist image: %w", err),
		)
		return
	}
	err = WriteJSON(w, http.StatusCreated, api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	bimage, err := h.artistService.GetImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(
				w, http.StatusNotFound,
				fmt.Errorf("artist image not found or artist doesn't exist: %w", err),
			)
		} else {
			_ = WriteError(
				w, http.StatusInternalServerError,
				fmt.Errorf("can't get artist image: %w", err),
			)
		}
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	_, _ = w.Write(bimage)
}
