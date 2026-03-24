package handlers

import (
	"encoding/json"
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
	w.Header().Set("Content-Type", "application/json")
	// if there's no artist_image it's still ok
	_, fileHeader, _ := r.FormFile("artist_image")
	artistName := r.FormValue("artist_name")
	if artistName == "" {
		http.Error(
			w,
			"can't find artistName in multipart-form or the name is empty",
			http.StatusBadRequest,
		)
		return
	}

	artistResponse, err := h.artistService.Create(r.Context(), artistName, fileHeader)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't create new artist: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(artistResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	w.Header().Set("Content-Type", "application/json")
	response, err := h.artistService.Delete(r.Context(), artistID)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't delete artist: %v", err.Error()), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistID int32) {
	w.Header().Set("Content-Type", "application/json")
	response, err := h.artistService.Get(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			http.Error(w, "Artist with this id is not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("can't get artist: %v", err.Error()), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	w.Header().Set("Content-Type", "application/json")
	err := h.artistService.DeleteImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			http.Error(w, "no artist with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	w.Header().Set("Content-Type", "application/json")
	err := h.artistService.UploadImage(r.Context(), artistID, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistID int32) {
	w.Header().Set("Content-Type", "image/webp")
	bimage, err := h.artistService.GetImage(r.Context(), artistID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			http.Error(w, "no artist with this id or no image", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	_, _ = w.Write(bimage)
}
