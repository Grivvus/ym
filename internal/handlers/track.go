package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
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
		_ = WriteError(w, http.StatusBadRequest, err)
		return
	}
	resp, err := h.trackService.UploadTrack(r.Context(), uploadParams, header)
	if err != nil {
		if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = WriteError(w, http.StatusConflict, err)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusAccepted, resp)
	if err != nil {
		slog.Error("Trackhandlers.UploadTrack, can't encode response", "err", err)
	}
}

func (h TrackHandlers) DeleteTrack(w http.ResponseWriter, r *http.Request, trackId int32) {
	err := h.trackService.DeleteTrack(r.Context(), trackId)
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
}

func (h TrackHandlers) GetTrackMeta(w http.ResponseWriter, r *http.Request, trackId int32) {
	metadata, err := h.trackService.GetMeta(r.Context(), trackId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, err)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusOK, metadata)
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
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusOK, tracks)
	if err != nil {
		slog.Error("can't encode response", "err", err)
	}
}

func (h TrackHandlers) DownloadTrack(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.DownloadTrackParams,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	quality := trackQuality(params.Quality)
	track, err := h.trackService.GetDownload(r.Context(), userID, trackId, quality)
	if err != nil {
		h.handleTrackFileError(w, "can't download track", err)
		return
	}
	defer func() { _ = track.Reader.Close() }()

	setTrackFileHeaders(w, track, true)
	http.ServeContent(w, r, track.DownloadName, time.Time{}, track.Reader)
}

func (h TrackHandlers) DownloadTrackHead(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.DownloadTrackHeadParams,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	quality := trackQuality(params.Quality)
	track, err := h.trackService.GetDownloadMeta(r.Context(), userID, trackId, quality)
	if err != nil {
		h.handleTrackFileError(w, "can't fetch track download metadata", err)
		return
	}

	setTrackFileHeaders(w, track, true)
	w.WriteHeader(http.StatusOK)
}

func (h TrackHandlers) StreamTrack(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.StreamTrackParams,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	h.serveTrack(w, r, userID, trackId, trackQuality(params.Quality))
}

func (h TrackHandlers) StreamTrackHead(
	w http.ResponseWriter, r *http.Request,
	trackId int32, params api.StreamTrackHeadParams,
) {
	userID, ok := requireAuthenticatedUserID(w, r)
	if !ok {
		return
	}
	h.serveTrack(w, r, userID, trackId, trackQuality(params.Quality))
}

func (h TrackHandlers) serveTrack(
	w http.ResponseWriter, r *http.Request,
	userID, trackId int32, quality string,
) {
	stream, err := h.trackService.GetStream(r.Context(), userID, trackId, quality)
	if err != nil {
		h.handleTrackFileError(w, "can't stream track", err)
		return
	}
	defer func() { _ = stream.Reader.Close() }()

	setTrackFileHeaders(w, stream, false)
	http.ServeContent(w, r, stream.Name, time.Time{}, stream.Reader)
}

func (h TrackHandlers) handleTrackFileError(w http.ResponseWriter, msg string, err error) {
	h.logger.Error(msg, "err", err)
	if errors.Is(err, service.ErrBadParams) {
		_ = WriteError(w, http.StatusBadRequest, err)
	} else if errors.Is(err, service.ErrPresetCantBeSelected) {
		_ = WriteError(w, http.StatusBadRequest, fmt.Errorf("%w. Probably wrong name", err))
	} else if errors.Is(err, service.ErrUnauthorized) {
		_ = WriteError(w, http.StatusForbidden, err)
	} else if _, ok := errors.AsType[service.ErrNotFound](err); ok {
		_ = WriteError(w, http.StatusNotFound, err)
	} else {
		_ = WriteError(w, http.StatusInternalServerError, err)
	}
}

func setTrackFileHeaders(w http.ResponseWriter, track service.TrackStream, download bool) {
	if track.ContentType != "" && track.ContentType != "application/octet-stream" {
		w.Header().Set("Content-Type", track.ContentType)
	}
	if track.ContentLength >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(track.ContentLength, 10))
	}
	w.Header().Set("Accept-Ranges", "bytes")

	if !download {
		return
	}

	if etag := quotedETag(track.ETag); etag != "" {
		w.Header().Set("ETag", etag)
	}
	w.Header().Set(
		"Content-Disposition",
		mime.FormatMediaType("attachment", map[string]string{
			"filename": track.DownloadName,
		}),
	)
	w.Header().Set("X-Track-Quality-Requested", track.RequestedQuality)
	w.Header().Set("X-Track-Quality-Resolved", track.ResolvedQuality)
	if track.ChecksumSHA256 != "" {
		w.Header().Set("X-Track-Checksum-Sha256", track.ChecksumSHA256)
	}
}

func quotedETag(etag string) string {
	etag = strings.TrimSpace(etag)
	if etag == "" {
		return ""
	}
	return `"` + strings.Trim(etag, `"`) + `"`
}

func trackQuality(quality *string) string {
	if quality == nil || *quality == "" {
		return "standard"
	}
	return *quality
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
		IsGloballyAvailable: FormValueToBool(isGloballyAvailable),
		IsSingle:            FormValueToBool(isSingle),
	}

	return uploadParams, header, nil
}
