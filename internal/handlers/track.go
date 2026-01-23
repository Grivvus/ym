package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type TrackHandlers struct {
	trackService service.TrackService
}

func (h TrackHandlers) UploadTrack(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h TrackHandlers) DeleteTrack(w http.ResponseWriter, r *http.Request, trackId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h TrackHandlers) GetTrackMeta(w http.ResponseWriter, r *http.Request, trackId int) {
	w.WriteHeader(http.StatusNotImplemented)
}
