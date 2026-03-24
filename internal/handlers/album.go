package handlers

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type AlbumHandlers struct {
	albumService service.AlbumService
	logger       *slog.Logger
}

func (h AlbumHandlers) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	_, fileHeader, _ := r.FormFile("album_cover")
	artist := r.FormValue("artist_id")
	name := r.FormValue("album_name")
	if name == "" || artist == "" {
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("form fields are not set or empty"),
		)
		return
	}
	artistID, err := strconv.Atoi(artist)
	if err != nil {
		_ = writeError(w, http.StatusBadRequest, fmt.Errorf("artist_id must be int"))
		return
	}
	var params = service.AlbumCreateParams{
		ArtistID: int32(artistID),
		Name:     name,
	}

	albumResponse, err := h.albumService.Create(r.Context(), params, fileHeader)
	if err != nil {
		_ = writeError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create album, cause: %w ", err),
		)
		return
	}
	err = writeJSON(w, http.StatusCreated, albumResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) GetAlbum(w http.ResponseWriter, r *http.Request, albumId int32) {
	albumResp, err := h.albumService.Get(r.Context(), albumId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			_ = writeError(w, http.StatusNotFound, fmt.Errorf("can't find album with this id: %w", err))
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}
	err = writeJSON(w, http.StatusOK, albumResp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbum(w http.ResponseWriter, r *http.Request, albumId int32) {
	albumResp, err := h.albumService.Delete(r.Context(), albumId)
	if err != nil {
		_ = writeError(w, http.StatusInternalServerError, err)
		return
	}
	err = writeJSON(w, http.StatusOK, albumResp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbumCover(w http.ResponseWriter, r *http.Request, albumId int32) {
	err := h.albumService.DeleteCover(r.Context(), albumId)
	if err != nil {
		http.Error(w, "can't delete cover: "+err.Error(), http.StatusInternalServerError)
		return
	}
	err = writeJSON(w, http.StatusOK, api.AlbumCoverResponse{AlbumId: albumId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) GetAlbumCover(w http.ResponseWriter, r *http.Request, albumId int32) {
	w.Header().Set("Content-Type", "image/webp")
	bimage, err := h.albumService.GetCover(r.Context(), albumId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			_ = writeError(
				w, http.StatusNotFound,
				fmt.Errorf("no album with this id was found: %w", err),
			)
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	_, err = w.Write(bimage)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) UploadAlbumCover(w http.ResponseWriter, r *http.Request, albumId int32) {
	err := h.albumService.UploadCover(r.Context(), albumId, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no album with this id was found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = writeJSON(w, http.StatusCreated, api.AlbumCoverResponse{AlbumId: albumId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
