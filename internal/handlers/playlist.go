package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type PlaylistHandlers struct {
	playlistService service.PlaylistService
	logger          *slog.Logger
}

func (h PlaylistHandlers) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	params, coverFileHeader, err := h.parsePostParams(r, userID)
	if err != nil {
		_ = writeError(w, http.StatusBadRequest, err)
		return
	}
	playlistResponse, err := h.playlistService.Create(r.Context(), params, coverFileHeader)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = writeError(w, http.StatusConflict, fmt.Errorf("playlist with this name already exists"))
			return
		}
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
	w.WriteHeader(http.StatusNotImplemented)
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
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	playlists, err := h.playlistService.GetUserPlaylists(r.Context(), userID)
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
		if e, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = writeError(w, http.StatusConflict, e)
			return
		}
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	err := h.playlistService.DeleteCover(r.Context(), playlistId)
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
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(w, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, service.ErrBadParams) {
			_ = writeError(w, http.StatusBadRequest, err)
			return
		}
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

func (h PlaylistHandlers) parsePostParams(
	r *http.Request, userID int32,
) (service.PlaylistCreateParams, *multipart.FileHeader, error) {
	_, coverFileHeader, _ := r.FormFile("playlist_cover")

	playlistName := r.FormValue("playlist_name")
	isPublic := r.FormValue("is_public")
	if playlistName == "" {
		return service.PlaylistCreateParams{}, nil, fmt.Errorf("playlist_name is required")
	}
	params := service.PlaylistCreateParams{
		OwnerID:  userID,
		IsPublic: formValueToBool(isPublic),
		Name:     playlistName,
	}

	return params, coverFileHeader, nil
}
