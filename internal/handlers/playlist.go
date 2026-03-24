package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
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
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("form fields are not set or empty"))
		return
	}

	ownerID, err := strconv.Atoi(owner)
	if err != nil {
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("owner_id must be int"))
		return
	}
	params.Name = playlistName
	params.OwnerID = int32(ownerID)

	playlistResponse, err := h.playlistService.Create(r.Context(), params, coverFileHeader)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create playlist: %w", err),
		)
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
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete playlist: %w", err),
		)
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
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(w, http.StatusNotFound, fmt.Errorf("can't find playlist with this id"))
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
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
		_ = writeError(w, http.StatusInternalServerError, err)
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
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}
	err = h.playlistService.AddTrack(r.Context(), playlistID, body.TrackId)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	err := h.playlistService.Delete(r.Context(), playlistId)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete cover: %w", err),
		)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) GetPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	bimage, err := h.playlistService.GetCover(r.Context(), playlistId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(w, http.StatusNotFound, fmt.Errorf("can't find playlist with this id"))
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
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
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't upload image: %w", err),
		)
		return
	}
	err = writeJSON(w, http.StatusCreated, api.PlaylistCoverResponse{PlaylistId: playlistId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
