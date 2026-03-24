package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type ArtistHandlers struct {
	artistService service.ArtistService
	logger        *slog.Logger
}

func (h ArtistHandlers) CreateArtist(w http.ResponseWriter, r *http.Request) {
	// if there's no artist_image it's still ok
	_, fileHeader, _ := r.FormFile("artist_image")
	artistName := r.FormValue("artist_name")
	if artistName == "" {
		_ = writeError(w, http.StatusBadRequest, errors.New("artist name is required"))
		return
	}

	artistResponse, err := h.artistService.Create(r.Context(), artistName, fileHeader)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create new artist: %w", err),
		)
		return
	}

	err = writeJSON(w, http.StatusCreated, artistResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	response, err := h.artistService.Delete(r.Context(), artistID)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete artist: %w", err),
		)
		return
	}
	err = writeJSON(w, http.StatusOK, response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	response, err := h.artistService.Get(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find artist with this id: %w", err),
			)
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	err = writeJSON(w, http.StatusOK, response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	err := h.artistService.DeleteImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find artist with this id: %w", err),
			)
		} else {
			_ = writeError(
				w, http.StatusInternalServerError,
				fmt.Errorf("can't delete artist image: %w", err),
			)
		}
		return
	}
	err = writeJSON(w, http.StatusOK, api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	err := h.artistService.UploadImage(r.Context(), artistID, r.Body)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't upload artist image: %w", err),
		)
		return
	}
	err = writeJSON(w, http.StatusCreated, api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	bimage, err := h.artistService.GetImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(
				w, http.StatusNotFound,
				fmt.Errorf("artist image not found or artist doesn't exist: %w", err),
			)
		} else {
			_ = writeError(
				w, http.StatusInternalServerError,
				fmt.Errorf("can't get artist image: %w", err),
			)
		}
		return
	}

	w.Header().Set("Content-Type", "image/webp")
	_, _ = w.Write(bimage)
}
