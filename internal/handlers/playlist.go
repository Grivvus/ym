package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type PlaylistHandlers struct {
	playlistService service.PlaylistService
}

func (h PlaylistHandlers) CreatePlaylist(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h PlaylistHandlers) DeletePlaylist(w http.ResponseWriter, r *http.Request, playlistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h PlaylistHandlers) GetPlaylist(w http.ResponseWriter, r *http.Request, playlistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h PlaylistHandlers) DeletePlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h PlaylistHandlers) GetPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h PlaylistHandlers) UploadPlaylistCover(w http.ResponseWriter, r *http.Request, playlistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}
