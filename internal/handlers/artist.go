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
	var artist api.ArtistCreateRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&artist)
	if err != nil {
		http.Error(w, "failed to decode json body", http.StatusBadRequest)
		return
	}

	artistResponse, err := h.artistService.Create(ctx, artist)
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

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistId int) {
	ctx := context.TODO()
	response, err := h.artistService.Delete(ctx, artistId)
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

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistId int) {
	ctx := context.TODO()
	response, err := h.artistService.Get(ctx, artistId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "Artist with this id is not found", http.StatusNotFound)
		} else {
			http.Error(w, fmt.Sprintf("can't delete artist: %v", err.Error()), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		slog.Error("ArtistHandlers.GetArtist", "err", err)
	}
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}
