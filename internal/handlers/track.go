package handlers

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type TrackHandlers struct {
	trackService service.TrackService
	logger       *slog.Logger
}

func (h TrackHandlers) UploadTrack(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	_, header, err := r.FormFile("track")
	if err != nil || header == nil {
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("form must include track file"))
		return
	}
	name := r.FormValue("name")
	album := r.FormValue("album_id")
	artist := r.FormValue("artist_id")
	isGloballyAvailable := r.FormValue("is_globally_available")
	isSingle := r.FormValue("is_single")
	if name == "" || artist == "" {
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("track name and artist are required"))
		return
	}
	artistID, err := strconv.Atoi(artist)
	if err != nil {
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("artist_id must be valid int"))
		return
	}
	var albumID *int32
	if album != "" {
		albumIDInt, err := strconv.Atoi(album)
		if err != nil {
			_ = writeError(w, http.StatusBadRequest, fmt.Errorf("album_id must be valid int"))
			return
		}
		albumID32 := int32(albumIDInt)
		albumID = &albumID32
	}
	var uploadParams = service.TrackUploadParams{
		ArtistID:            int32(artistID),
		AlbumID:             albumID,
		Name:                name,
		UploadBy:            &userID,
		IsGloballyAvailable: formValueToBool(isGloballyAvailable),
		IsSingle:            formValueToBool(isSingle),
	}

	resp, err := h.trackService.UploadTrack(r.Context(), uploadParams, header)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = writeJSON(w, http.StatusCreated, resp)
	if err != nil {
		slog.Error("Trackhandlers.UploadTrack, can't encode response", "err", err)
	}
}

func (h TrackHandlers) DeleteTrack(w http.ResponseWriter, r *http.Request, trackId int32) {
	err := h.trackService.DeleteTrack(r.Context(), trackId)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
}

func (h TrackHandlers) GetTrackMeta(w http.ResponseWriter, r *http.Request, trackId int32) {
	metadata, err := h.trackService.GetMeta(r.Context(), trackId)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = writeJSON(w, http.StatusOK, metadata)
	if err != nil {
		slog.Error("TrackHandlers.GetTrackMeta, can't encode response", "err", err)
	}
}

func (h TrackHandlers) GetTracks(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	tracks, err := h.trackService.GetUserTracks(r.Context(), userID)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = writeJSON(w, http.StatusOK, tracks)
	if err != nil {
		slog.Error("can't encode response", "err", err)
	}
}

func (h TrackHandlers) StreamTrack(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.StreamTrackParams,
) {

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	var quality string
	if params.Quality == nil {
		quality = "standard"
	} else {
		quality = *params.Quality
	}

	meta, err := h.trackService.GetStreamMeta(r.Context(), trackId, quality)
	if err != nil {
		if errors.Is(err, service.ErrPresetCantBeSelected) {
			_ = writeError(w, http.StatusBadRequest, fmt.Errorf("%w. Probably wrong name", err))
		} else if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(w, http.StatusNotFound, err)
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	stream, err := h.trackService.GetStream(r.Context(), trackId, quality)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}

	_, err = io.Copy(w, stream)
	if err != nil {
		h.logger.Error("can't write stream to response", "err", err)
		_ = writeError(w, http.StatusInternalServerError, err)
	}
	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(int(meta.ContentLength)))
}

func (h TrackHandlers) StreamTrackHead(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.StreamTrackHeadParams,
) {
	var quality string
	if params.Quality == nil {
		quality = "standard"
	} else {
		quality = *params.Quality
	}

	meta, err := h.trackService.GetStreamMeta(r.Context(), trackId, quality)
	if err != nil {
		if errors.Is(err, service.ErrPresetCantBeSelected) {
			_ = writeError(w, http.StatusBadRequest, fmt.Errorf("%w. Probably wrong name", err))
		} else if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(w, http.StatusNotFound, err)
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(int(meta.ContentLength)))
}

func formValueToBool(val string) bool {
	return val == "true" || val == "True"
}
