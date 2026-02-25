package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type AlbumHandlers struct {
	albumService service.AlbumService
}

func (h AlbumHandlers) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
	_, fileHeader, _ := r.FormFile("album_cover")
	artist := r.FormValue("artist_id")
	name := r.FormValue("album_name")
	if name == "" || artist == "" {
		http.Error(w, "Form fields are not set or empty", http.StatusBadRequest)
		return
	}
	artistID, err := strconv.Atoi(artist)
	if err != nil {
		http.Error(w, "artist_id must be int", http.StatusBadRequest)
		return
	}
	var params = service.AlbumCreateParams{
		ArtistID: artistID,
		Name:     name,
	}

	albumResponse, err := h.albumService.Create(ctx, params, fileHeader)
	if err != nil {
		http.Error(w, "can't create album, cause: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(albumResponse)
	if err != nil {
		slog.Error("AlbumHandlers.CreateAlbum: can't encode response", "err", err)
	}
}

func (h AlbumHandlers) GetAlbum(w http.ResponseWriter, r *http.Request, albumId int) {
	ctx := context.TODO()
	albumResp, err := h.albumService.Get(ctx, albumId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "can't find album with this id", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = json.NewEncoder(w).Encode(albumResp)
	if err != nil {
		slog.Error("AlbumHandlers.GetAlbum: can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbum(w http.ResponseWriter, r *http.Request, albumId int) {
	ctx := context.TODO()
	albumResp, err := h.albumService.Delete(ctx, albumId)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	err = json.NewEncoder(w).Encode(albumResp)
	if err != nil {
		slog.Error("ALbumHandlers.DeleteAlbum: can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	ctx := context.TODO()
	err := h.albumService.DeleteCover(ctx, albumId)
	if err != nil {
		http.Error(w, "can't delete cover: "+err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(api.AlbumCoverResponse{AlbumId: albumId})
}

func (h AlbumHandlers) GetAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	ctx := context.TODO()
	bimage, err := h.albumService.GetCover(ctx, albumId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no album with this id was found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(bimage)
}

func (h AlbumHandlers) UploadAlbumCover(w http.ResponseWriter, r *http.Request, albumId int) {
	ctx := context.TODO()
	err := h.albumService.UploadCover(ctx, albumId, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no album with this id was found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(api.AlbumCoverResponse{AlbumId: albumId})
}
