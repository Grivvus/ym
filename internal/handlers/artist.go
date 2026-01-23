package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/service"
)

type ArtistHandlers struct {
	artistService service.ArtistService
}

func (h ArtistHandlers) CreateArtist(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) DeleteArtist(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) GetArtist(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) DeleteArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) GetArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h ArtistHandlers) UploadArtistImage(w http.ResponseWriter, r *http.Request, artistId int) {
	w.WriteHeader(http.StatusNotImplemented)
}
