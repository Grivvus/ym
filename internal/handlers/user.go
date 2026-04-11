package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type UserHandlers struct {
	userService service.UserService
	logger      *slog.Logger
}

func (u UserHandlers) GetUserById(
	w http.ResponseWriter, r *http.Request, userId int32,
) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	user, err := u.userService.GetUserByID(r.Context(), userId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusBadRequest, fmt.Errorf("can't find user with this id"))
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, user)
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}

func (u UserHandlers) ChangeUser(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	var toUpdate api.UserUpdate
	err := json.NewDecoder(r.Body).Decode(&toUpdate)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}
	resp, err := u.userService.ChangeUser(r.Context(), userId, toUpdate)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, fmt.Errorf("no such user"))
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		u.logger.Error("unexpected error", "err", err)
	}
}

func (u UserHandlers) ChangePassword(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	var updatePassword api.UserChangePassword
	err := json.NewDecoder(r.Body).Decode(&updatePassword)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}
	err = u.userService.ChangePassword(r.Context(), userId, updatePassword)
	if err != nil {
		u.logger.Error("can't change password", "err", err)
		if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, err)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
}

func (u UserHandlers) UploadUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	err := u.userService.UploadAvatar(r.Context(), userId, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, fmt.Errorf("no such user"))
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusCreated, api.MessageResponse{Msg: "avatar was uploaded"})
	if err != nil {
		u.logger.Error("can't encode json", "err", err)
	}
}

func (u UserHandlers) GetUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	img, err := u.userService.GetAvatar(r.Context(), userId)
	if err != nil {
		if _, t := errors.AsType[service.ErrNotFound](err); t {
			_ = WriteError(w, http.StatusNotFound, fmt.Errorf("user not found or no avatar"))
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	_, err = w.Write(img)
	if err != nil {
		u.logger.Error("can't write response")
	}
}

func (u UserHandlers) DeleteUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	err := u.userService.DeleteAvatar(r.Context(), userId)
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	err = WriteJSON(w, http.StatusOK, api.MessageResponse{Msg: "avatar was deleted"})
	if err != nil {
		u.logger.Error("can't encode json", "err", err)
	}
}
