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

func (u UserHandlers) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users, err := u.userService.GetAllUsers(r.Context())
	if err != nil {
		MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		return
	}
	errSend := WriteJSON(w, http.StatusOK, users)
	if errSend != nil {
		u.logger.Error("can't encode response", "err", errSend)
	}
}

func (u UserHandlers) GetUser(
	w http.ResponseWriter, r *http.Request, userId int32,
) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	user, err := u.userService.GetUser(r.Context(), userId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusBadRequest, fmt.Errorf("can't find user with this id"), u.logger)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, user)
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}

func (u UserHandlers) GetSimpleUser(w http.ResponseWriter, r *http.Request, userId int32) {
	user, err := u.userService.GetUser(r.Context(), userId)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusBadRequest, fmt.Errorf("can't find user with this id"), u.logger)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		}
		return
	}
	simpleUser := api.SimpleUser{
		Id:       userId,
		Username: user.Username,
	}
	errSend := WriteJSON(w, http.StatusOK, simpleUser)
	if errSend != nil {
		u.logger.Error("can't encode response", "err", errSend)
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
			MustWriteError(w, http.StatusNotFound, fmt.Errorf("no such user"), u.logger)
			return
		}
		if errors.Is(err, service.ErrBadParams) {
			MustWriteError(w, http.StatusBadRequest, err, u.logger)
			return
		}
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			MustWriteError(w, http.StatusConflict, fmt.Errorf("user already exists"), u.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}

func (u UserHandlers) ChangePassword(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	var updatePassword api.UserChangePassword
	err := json.NewDecoder(r.Body).Decode(&updatePassword)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), u.logger,
		)
		return
	}
	err = u.userService.ChangePassword(r.Context(), userId, updatePassword)
	if err != nil {
		u.logger.Warn("can't change password", "err", err)
		if errors.Is(err, service.ErrBadParams) {
			MustWriteError(w, http.StatusBadRequest, err, u.logger)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, err, u.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, u.logger)
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
			MustWriteError(w, http.StatusBadRequest, err, u.logger)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(w, http.StatusNotFound, fmt.Errorf("no such user"), u.logger)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		return
	}
	err = WriteJSON(w, http.StatusCreated, api.MessageResponse{Msg: "avatar was uploaded"})
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}

func (u UserHandlers) GetUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	img, err := u.userService.GetAvatar(r.Context(), userId)
	if err != nil {
		if _, t := errors.AsType[service.ErrNotFound](err); t {
			MustWriteError(w, http.StatusNotFound, fmt.Errorf("user not found or no avatar"), u.logger)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		}
		return
	}
	w.Header().Set("Content-Type", "image/webp")
	_, err = w.Write(img)
	if err != nil {
		u.logger.Error("can't write response", "err", err)
	}
}

func (u UserHandlers) DeleteUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	if !requireCurrentUser(w, r, userId) {
		return
	}
	err := u.userService.DeleteAvatar(r.Context(), userId)
	if err != nil {
		MustWriteError(w, http.StatusInternalServerError, err, u.logger)
		return
	}
	err = WriteJSON(w, http.StatusOK, api.MessageResponse{Msg: "avatar was deleted"})
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}
