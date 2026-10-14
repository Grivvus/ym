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
		MustWriteError(w, http.StatusBadRequest, err, h.logger)
		return
	}
	playlistResponse, err := h.playlistService.Create(r.Context(), params, coverFileHeader)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			MustWriteError(
				w, http.StatusConflict,
				fmt.Errorf("playlist with this name already exists"),
				h.logger,
			)
			return
		}
		MustWriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create playlist: %w", err), h.logger,
		)
		return
	}
	err = WriteJSON(w, http.StatusCreated, playlistResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) UpdatePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	userID, _ := requireAuthenticatedUserID(w, r)
	var newData api.PlaylistUpdateRequest
	err := json.NewDecoder(r.Body).Decode(&newData)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("can't decode json body"))
		return
	}
	updatedPlaylist, err := h.playlistService.ChangePlaylist(r.Context(), userID, playlistId, newData)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			WriteError(w, http.StatusForbidden, err)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			WriteError(w, http.StatusNotFound, err)
			return
		}
		WriteError(w, http.StatusInternalServerError, err)
		return
	}
	WriteJSON(w, http.StatusOK, updatedPlaylist)
}

func (h PlaylistHandlers) DeletePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	userID, _ := requireAuthenticatedUserID(w, r)
	err := h.playlistService.Delete(r.Context(), playlistId, userID)
	if err != nil {
		MustWriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete playlist: %w", err), h.logger,
		)
		return
	}
	err = WriteJSON(w, http.StatusOK, api.PlaylistDeleteResponse{PlaylistId: playlistId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) GetPlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	playlistInfo, err := h.playlistService.Get(r.Context(), userID, playlistId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find playlist with this id"), h.logger,
			)
		} else if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(w, http.StatusForbidden, err, h.logger)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		}
		return
	}
	err = WriteJSON(w, http.StatusOK, playlistInfo)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h PlaylistHandlers) GetPlaylists(
	w http.ResponseWriter, r *http.Request,
	params api.GetPlaylistsParams,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	playlists, err := h.playlistService.GetUserPlaylists(r.Context(), userID, params)
	if err != nil {
		MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		return
	}
	err = WriteJSON(w, http.StatusOK, playlists)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
func (h PlaylistHandlers) AddTrackToPlaylist(
	w http.ResponseWriter, r *http.Request, playlistID int32,
) {
	userID, _ := requireAuthenticatedUserID(w, r)
	var body api.AddTrackToPlaylistJSONBody
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}
	err = h.playlistService.AddTrack(r.Context(), playlistID, userID, body.TrackId)
	if err != nil {
		if e, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			MustWriteError(w, http.StatusConflict, e, h.logger)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, err, h.logger)
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(w, http.StatusForbidden, err, h.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		return
	}
}

func (h PlaylistHandlers) PutTrackToPlaylist(
	w http.ResponseWriter, r *http.Request, playlistID int32, trackID int32,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	err := h.playlistService.EnsureTrack(r.Context(), playlistID, userID, trackID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, err, h.logger)
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(w, http.StatusForbidden, err, h.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) DeleteTrackFromPlaylist(
	w http.ResponseWriter, r *http.Request, playlistID int32, trackID int32,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	err := h.playlistService.DeleteTrack(r.Context(), playlistID, userID, trackID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, err, h.logger)
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(w, http.StatusForbidden, err, h.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) SharePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	ownerID, _ := requireAuthenticatedUserID(w, r)
	var body api.PlaylistShareRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	err = h.playlistService.SharePlaylistWithUsers(r.Context(), playlistId, ownerID, body)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			WriteError(w, http.StatusForbidden, err)
			return
		}
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			WriteError(w, http.StatusConflict, err)
			return
		}
		WriteError(w, http.StatusInternalServerError, err)
		return
	}
	WriteJSON(w, http.StatusOK, api.MessageResponse{
		Msg: "playlist was shared with some users",
	})
}

func (h PlaylistHandlers) RevokePlaylist(w http.ResponseWriter, r *http.Request, playlistId int32) {
	ownerID, _ := requireAuthenticatedUserID(w, r)
	var body api.PlaylistRevokeAccessRequest
	err := json.NewDecoder(r.Body).Decode(&body)
	if err != nil {
		WriteError(w, http.StatusBadRequest, fmt.Errorf("bad json: %w", err))
		return
	}
	err = h.playlistService.RevokePlaylistAccess(r.Context(), playlistId, ownerID, body.UserId)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			WriteError(w, http.StatusForbidden, err)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			WriteError(w, http.StatusNotFound, err)
			return
		}
		WriteError(w, http.StatusInternalServerError, err)
		return
	}
	WriteJSON(w, http.StatusOK, api.MessageResponse{
		Msg: "playlist access was revoked from this user",
	})
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	userID, _ := requireAuthenticatedUserID(w, r)
	err := h.playlistService.DeleteCover(r.Context(), playlistId, userID)
	if err != nil {
		MustWriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete cover: %w", err), h.logger,
		)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h PlaylistHandlers) GetPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int32) {
	bimage, err := h.playlistService.GetCover(r.Context(), playlistId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(
				w, http.StatusNotFound,
				fmt.Errorf("can't find playlist with this id"), h.logger,
			)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
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
	userID, _ := requireAuthenticatedUserID(w, r)
	err := h.playlistService.UploadCover(r.Context(), userID, playlistId, r.Body)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, err, h.logger)
			return
		}
		if errors.Is(err, service.ErrBadParams) {
			MustWriteError(w, http.StatusBadRequest, err, h.logger)
			return
		}
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(w, http.StatusForbidden, err, h.logger)
		}
		MustWriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't upload image: %w", err), h.logger,
		)
		return
	}
	err = WriteJSON(w, http.StatusCreated, api.PlaylistCoverResponse{PlaylistId: playlistId})
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
		IsPublic: FormValueToBool(isPublic),
		Name:     playlistName,
	}

	return params, coverFileHeader, nil
}
