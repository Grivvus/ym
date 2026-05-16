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

type AlbumHandlers struct {
	albumService service.AlbumService
	authService  service.AuthService
	logger       *slog.Logger
}

func (h AlbumHandlers) CreateAlbum(w http.ResponseWriter, r *http.Request) {
	params, fileHeader, err := h.parsePostForm(r)
	if err != nil {
		_ = WriteError(w, http.StatusBadRequest, err)
		return
	}
	albumResponse, err := h.albumService.Create(r.Context(), params, fileHeader)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = WriteError(w, http.StatusConflict, fmt.Errorf(
				"%w: this artist already has album with this name", err,
			))
			return
		}
		_ = WriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't create album, cause: %w ", err),
		)
		return
	}
	err = WriteJSON(w, http.StatusCreated, albumResponse)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) GetAlbum(w http.ResponseWriter, r *http.Request, albumId int32) {
	albumResp, err := h.albumService.Get(r.Context(), albumId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, fmt.Errorf("can't find album with this id: %w", err))
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}
	err = WriteJSON(w, http.StatusOK, albumResp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbum(w http.ResponseWriter, r *http.Request, albumId int32) {
	ok := requireSuperuser(w, r, h.authService)
	if !ok {
		return
	}

	albumResp, err := h.albumService.Delete(r.Context(), albumId)
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusOK, albumResp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) DeleteAlbumCover(w http.ResponseWriter, r *http.Request, albumId int32) {
	err := h.albumService.DeleteCover(r.Context(), albumId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, err)
			return
		} else if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		_ = WriteError(
			w, http.StatusInternalServerError,
			fmt.Errorf("can't delete cover: %w", err),
		)
		return
	}
	err = WriteJSON(w, http.StatusOK, api.AlbumCoverResponse{AlbumId: albumId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) GetAlbumCover(w http.ResponseWriter, r *http.Request, albumId int32) {
	w.Header().Set("Content-Type", "image/webp")
	bimage, err := h.albumService.GetCover(r.Context(), albumId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(
				w, http.StatusNotFound,
				fmt.Errorf("no album with this id was found: %w", err),
			)
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
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
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, fmt.Errorf("no album with this id was found"))
		} else if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}
	err = WriteJSON(w, http.StatusCreated, api.AlbumCoverResponse{AlbumId: albumId})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AlbumHandlers) parsePostForm(
	r *http.Request,
) (service.AlbumCreateParams, *multipart.FileHeader, error) {
	_, fileHeader, _ := r.FormFile("album_cover")
	artist := r.FormValue("artist_id")
	name := r.FormValue("album_name")
	year := r.FormValue("release_year")
	date := r.FormValue("release_full_date")
	if name == "" || artist == "" {
		return service.AlbumCreateParams{}, nil, fmt.Errorf("form fields are not set or empty")
	}
	artistID, err := strconv.Atoi(artist)
	if err != nil {
		return service.AlbumCreateParams{}, nil, fmt.Errorf("artist_id must be int")
	}
	var params = service.AlbumCreateParams{
		ArtistID: int32(artistID),
		Name:     name,
	}
	if year == "" {
		params.ReleaseYear = nil
	} else {
		yearNum, err := strconv.ParseInt(year, 10, 32)
		if err != nil {
			return service.AlbumCreateParams{}, nil, fmt.Errorf("year must be integer")
		}
		i32 := int32(yearNum)
		params.ReleaseYear = &i32
	}
	if date == "" {
		params.ReleaseDate = nil
	} else {
		dt, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return service.AlbumCreateParams{}, nil, fmt.Errorf("invalid date format, expected RFC3339")
		}
		params.ReleaseDate = &dt
	}
	return params, fileHeader, nil
}
