package handlers

import (
	"encoding/json"
	"errors"
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
	w.Header().Set("Content-Type", "application/json")
	user, err := u.userService.GetUserByID(r.Context(), userId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "Wrong username", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(user)
	if err != nil {
		u.logger.Error("can't encode response", "err", err)
	}
}

func (u UserHandlers) ChangeUser(w http.ResponseWriter, r *http.Request, userId int32) {
	w.Header().Set("Content-Type", "application/json")
	var toUpdate api.UserUpdate
	err := json.NewDecoder(r.Body).Decode(&toUpdate)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	resp, err := u.userService.ChangeUser(r.Context(), userId, toUpdate)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "No such user", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		u.logger.Error("unexpected error", "err", err)
	}
}

func (u UserHandlers) ChangePassword(w http.ResponseWriter, r *http.Request, userId int32) {
	w.Header().Set("Content-Type", "application/json")
	var updatePassword api.UserChangePassword
	err := json.NewDecoder(r.Body).Decode(&updatePassword)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	err = u.userService.ChangePassword(r.Context(), userId, updatePassword)
	if err != nil {
		u.logger.Error("can't change password", "err", err)
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "old password is wrong", http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (u UserHandlers) UploadUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	w.Header().Set("Content-Type", "application/json")
	err := u.userService.UploadAvatar(r.Context(), userId, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no such user", http.StatusNotFound)
		} else {
			http.Error(w, "unknown server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(api.MessageResponse{Msg: "avatar was uploaded"})
	if err != nil {
		u.logger.Error("can't encode json", "err", err)
	}
}

func (u UserHandlers) GetUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	w.Header().Set("Content-Type", "image/webp")
	img, err := u.userService.GetAvatar(r.Context(), userId)
	if err != nil {
		if _, t := errors.AsType[service.ErrNotFound](err); t {
			http.Error(w, "user not found or no avatar", http.StatusNotFound)
		} else {
			http.Error(w, "unknown server error", http.StatusInternalServerError)
		}
		return
	}
	_, err = w.Write(img)
	if err != nil {
		u.logger.Error("can't send response")
	}
}

func (u UserHandlers) DeleteUserAvatar(w http.ResponseWriter, r *http.Request, userId int32) {
	w.Header().Set("Content-Type", "application/json")
	err := u.userService.DeleteAvatar(r.Context(), userId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no such user", http.StatusNotFound)
		} else {
			http.Error(w, "unknown server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = json.NewEncoder(w).Encode(api.MessageResponse{Msg: "avatar was deleted"})
	if err != nil {
		u.logger.Error("can't encode json", "err", err)
	}
}
