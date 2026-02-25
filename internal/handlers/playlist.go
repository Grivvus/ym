package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type PlaylistHandlers struct {
	playlistService service.PlaylistService
}

func (h PlaylistHandlers) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
	_, coverFileHeader, _ := r.FormFile("playlist_cover")

	var params service.PlaylistCreateParams
	owner := r.FormValue("owner_id")
	playlistName := r.FormValue("playlist_name")
	if owner == "" || playlistName == "" {
		http.Error(w, "Form fields are not set or empty", http.StatusBadRequest)
		return
	}

	ownerID, err := strconv.Atoi(owner)
	if err != nil {
		http.Error(w, "owner_id must be int", http.StatusBadRequest)
		return
	}
	params.Name = playlistName
	params.OwnerID = ownerID

	playlistResponse, err := h.playlistService.Create(ctx, params, coverFileHeader)
	if err != nil {
		http.Error(w, "can't create playlist: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(playlistResponse)
	if err != nil {
		slog.Error("PlaylistHandlers.CraetePlaylist, can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) DeletePlaylist(w http.ResponseWriter, r *http.Request, playlistId int) {
	ctx := context.TODO()
	err := h.playlistService.Delete(ctx, playlistId)
	if err != nil {
		http.Error(w, "can't delete playlist: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(api.PlaylistDeleteResponse{PlaylistId: playlistId})
	if err != nil {
		slog.Error("PlaylistHandlers.DeletePlaylist: can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) GetPlaylist(w http.ResponseWriter, r *http.Request, playlistId int) {
	ctx := context.TODO()
	playlistInfo, err := h.playlistService.Get(ctx, playlistId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "can't find playlist with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = json.NewEncoder(w).Encode(playlistInfo)
	if err != nil {
		slog.Error("PlaylistHandlers.GetPlaylist: can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	ctx := context.TODO()
	err := h.playlistService.Delete(ctx, playlistId)
	if err != nil {
		http.Error(w, "can't delete cover: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) GetPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	ctx := context.TODO()
	bimage, err := h.playlistService.GetCover(ctx, playlistId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "can't find playlist with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	_, _ = w.Write(bimage)
}

func (h PlaylistHandlers) UploadPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	ctx := context.TODO()
	err := h.playlistService.UploadCover(ctx, playlistId, r.Body)
	if err != nil {
		http.Error(w, "can't upload image: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(api.PlaylistCoverResponse{PlaylistId: playlistId})
	if err != nil {
		slog.Error("PlaylistHandlers.UploadPlaylistCover: can't encode response", "err", err)
	}
}
