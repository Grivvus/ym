package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type TrackHandlers struct {
	trackService service.TrackService
}

func (h TrackHandlers) UploadTrack(w http.ResponseWriter, r *http.Request) {
	var uploadData api.TrackUploadRequest
	err := json.NewDecoder(r.Body).Decode(&uploadData)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	resp, err := h.trackService.UploadTrack(context.TODO(), uploadData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("Trackhandlers.UploadTrack, can't encode response", "err", err)
	}
}

func (h TrackHandlers) DeleteTrack(w http.ResponseWriter, r *http.Request, trackId int) {
	err := h.trackService.DeleteTrack(context.TODO(), trackId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h TrackHandlers) GetTrackMeta(w http.ResponseWriter, r *http.Request, trackId int) {
	metadata, err := h.trackService.GetMeta(context.TODO(), trackId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(metadata)
	if err != nil {
		slog.Error("TrackHandlers.GetTrackMeta, can't encode response", "err", err)
	}
}

func (h TrackHandlers) StreamTrack(
	w http.ResponseWriter, r *http.Request,
	trackId int, params api.StreamTrackParams,
) {

	rangeHeader := r.Header.Get("Range")
	if rangeHeader != "" {
		w.WriteHeader(http.StatusNotImplemented)
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}

	ctx := context.TODO()
	var quality string
	if params.Quality == nil {
		quality = "standard"
	} else {
		quality = *params.Quality
	}

	meta, err := h.trackService.GetStreamMeta(ctx, trackId, quality)
	if err != nil {
		if errors.Is(err, service.ErrPresetCantBeSelected) {
			http.Error(w, err.Error()+". Probably wrong name", http.StatusBadRequest)
		} else if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(int(meta.ContentLength)))

	stream, err := h.trackService.GetStream(ctx, trackId, quality)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(w, stream)
	if err != nil {
		slog.Error("Can't write stream to response", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (h TrackHandlers) StreamTrackHead(
	w http.ResponseWriter, r *http.Request,
	trackId int, params api.StreamTrackHeadParams,
) {
	var quality string
	if params.Quality == nil {
		quality = "standard"
	} else {
		quality = *params.Quality
	}

	meta, err := h.trackService.GetStreamMeta(context.TODO(), trackId, quality)
	if err != nil {
		if errors.Is(err, service.ErrPresetCantBeSelected) {
			http.Error(w, err.Error()+". Probably wrong name", http.StatusBadRequest)
		} else if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, err.Error(), http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.Header().Set("Content-Type", meta.ContentType)
	w.Header().Set("Content-Length", strconv.Itoa(int(meta.ContentLength)))
	w.WriteHeader(http.StatusOK)
}
