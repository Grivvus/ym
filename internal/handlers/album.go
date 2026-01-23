package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type AlbumHandlers struct {
	albumService service.AlbumService
}

func (h AlbumHandlers) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h AlbumHandlers) GetAlbum(w http.ResponseWriter, r *http.Request, albumId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h AlbumHandlers) DeleteAlbum(w http.ResponseWriter, r *http.Request, albumId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h AlbumHandlers) DeleteAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h AlbumHandlers) GetAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h AlbumHandlers) UploadAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	w.WriteHeader(http.StatusNotImplemented)
}
