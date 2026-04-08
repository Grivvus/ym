package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

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
	uploadParams, header, err := h.parsePostParams(r, userID)
	if err != nil {
		_ = writeError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := h.trackService.UploadTrack(r.Context(), uploadParams, header)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = writeJSON(w, http.StatusAccepted, resp)
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
	var quality string
	if params.Quality == nil {
		quality = "standard"
	} else {
		quality = *params.Quality
	}
	h.serveTrack(w, r, trackId, quality)
}

func (h TrackHandlers) serveTrack(
	w http.ResponseWriter, r *http.Request,
	trackId int32, quality string,
) {
	stream, err := h.trackService.GetStream(r.Context(), trackId, quality)
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
	defer func() { _ = stream.Reader.Close() }()

	if stream.ContentType != "" && stream.ContentType != "application/octet-stream" {
		w.Header().Set("Content-Type", stream.ContentType)
	}

	http.ServeContent(w, r, stream.Name, time.Time{}, stream.Reader)
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
	h.serveTrack(w, r, trackId, quality)
}

func (h TrackHandlers) parsePostParams(
	r *http.Request, userID int32,
) (service.TrackUploadParams, *multipart.FileHeader, error) {
	_, header, err := r.FormFile("track")
	if err != nil || header == nil {
		return service.TrackUploadParams{}, nil, fmt.Errorf("form must include track file")
	}
	name := r.FormValue("name")
	album := r.FormValue("album_id")
	artist := r.FormValue("artist_id")
	isGloballyAvailable := r.FormValue("is_globally_available")
	isSingle := r.FormValue("is_single")
	if name == "" || artist == "" {
		return service.TrackUploadParams{}, nil, fmt.Errorf("track name and artist are required")
	}
	artistID, err := strconv.Atoi(artist)
	if err != nil {
		return service.TrackUploadParams{}, nil, fmt.Errorf("artist_id must be valid int")
	}
	var albumID *int32
	if album != "" {
		albumIDInt, err := strconv.Atoi(album)
		if err != nil {
			return service.TrackUploadParams{}, nil, fmt.Errorf("album_id must be valid int")
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

	return uploadParams, header, nil
}
