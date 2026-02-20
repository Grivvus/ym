package handlers

import (
	"context"
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
}

func (h ArtistHandlers) CreateArtist(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
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

	artistResponse, err := h.artistService.Create(ctx, artistName, fileHeader)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't create new artist: %v", err.Error()), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(artistResponse)
	if err != nil {
		slog.Error("ArtistHandlers.CreateArtist", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistID int) {
	ctx := context.TODO()
	response, err := h.artistService.Delete(ctx, artistID)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't delete artist: %v", err.Error()), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("ArtistHandlers.DeleteArtist", "err", err)
	}
}

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistID int) {
	ctx := context.TODO()
	response, err := h.artistService.Get(ctx, artistID)
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
		slog.Error("ArtistHandlers.GetArtist", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistID int) {
	ctx := context.TODO()
	err := h.artistService.DeleteImage(ctx, artistID)
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
		slog.Error("ArtistHandlers.DeleteArtistImage, can't encode response", "err", err)
	}
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistID int) {
	ctx := context.TODO()
	err := h.artistService.UploadImage(ctx, artistID, r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(api.ArtistImageResponse{ArtistId: artistID})
	if err != nil {
		slog.Error("ArtistHandlers.UploadArtistImage, can't encode response", "err", err)
	}
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistID int) {
	ctx := context.TODO()
	bimage, err := h.artistService.GetImage(ctx, artistID)
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
