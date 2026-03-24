package handlers

import (
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
	logger          *slog.Logger
}

func (h PlaylistHandlers) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
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
	params.OwnerID = int32(ownerID)

	playlistResponse, err := h.playlistService.Create(r.Context(), params, coverFileHeader)
	if err != nil {
		http.Error(w, "can't create playlist: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = writeJSON(w, http.StatusCreated, playlistResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) UpdatePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	panic("implement me")
}

func (h PlaylistHandlers) DeletePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	err := h.playlistService.Delete(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "can't delete playlist: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = writeJSON(w, http.StatusOK, api.PlaylistDeleteResponse{PlaylistId: playlistId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) GetPlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	playlistInfo, err := h.playlistService.Get(r.Context(), playlistId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "can't find playlist with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = writeJSON(w, http.StatusOK, playlistInfo)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) GetPlaylists(w http.ResponseWriter, r *http.Request) {
	panic("not implemented")
	playlists, err := h.playlistService.GetUserPlaylists(r.Context(), 123)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = writeJSON(w, http.StatusOK, playlists)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
func (h PlaylistHandlers) AddTrackToPlaylist(
	w http.ResponseWriter, r *http.Request, playlistID int32,
) {
	var body api.AddTrackToPlaylistJSONBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = h.playlistService.AddTrack(r.Context(), playlistID, body.TrackId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	err := h.playlistService.Delete(r.Context(), playlistId)
	if err != nil {
		http.Error(w, "can't delete cover: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) GetPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	bimage, err := h.playlistService.GetCover(r.Context(), playlistId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "can't find playlist with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusNotFound)
		}
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	_, err = w.Write(bimage)
	if err != nil {
		h.logger.Error("can't write response", "err", err)
	}
}

func (h PlaylistHandlers) UploadPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	err := h.playlistService.UploadCover(r.Context(), playlistId, r.Body)
	if err != nil {
		http.Error(w, "can't upload image: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = writeJSON(w, http.StatusCreated, api.PlaylistCoverResponse{PlaylistId: playlistId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
