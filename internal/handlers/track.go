package handlers

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

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
