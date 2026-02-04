package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type UserHandlers struct {
	userService service.UserService
}

func (u UserHandlers) GetUserById(
	w http.ResponseWriter, r *http.Request, userId int,
) {
	ctx := context.TODO()
	user, err := u.userService.GetUserByID(ctx, userId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "Wrong username", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(user)
	if err != nil {
		slog.Error("Error while marshalling model", "err", err)
		http.Error(w, "", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		slog.Error("Unexpected error", "err", err)
	}
}

func (u UserHandlers) ChangeUser(w http.ResponseWriter, r *http.Request, userId int) {
	ctx := context.TODO()
	var toUpdate api.UserUpdate
	err := json.NewDecoder(r.Body).Decode(&toUpdate)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	resp, err := u.userService.ChangeUser(ctx, userId, toUpdate)
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
		slog.Error("Unexpected error", "err", err)
	}
}

func (u UserHandlers) ChangePassword(w http.ResponseWriter, r *http.Request, userId int) {
	ctx := context.TODO()
	var updatePassword api.UserChangePassword
	err := json.NewDecoder(r.Body).Decode(&updatePassword)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}
	err = u.userService.ChangePassword(ctx, userId, updatePassword)
	if err != nil {
		slog.Error("UserHandler.ChangePassword", "error", err)
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "Old password is wrong", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (u UserHandlers) UploadUserAvatar(w http.ResponseWriter, r *http.Request, userId int) {
	ctx := context.TODO()
	err := u.userService.UploadAvatar(ctx, userId, r.Body)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no such user", http.StatusNotFound)
		} else {
			http.Error(w, "unkown server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(api.MessageResponse{Msg: "avatar was uploaded"})
	if err != nil {
		slog.Error("UploadUserAvatar: can't encode json", "err", err)
	}
}

func (u UserHandlers) DeleteUserAvatar(w http.ResponseWriter, r *http.Request, userId int) {
	ctx := context.TODO()
	err := u.userService.DeleteAvatar(ctx, userId)
	if err != nil {
		if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "no such user", http.StatusNotFound)
		} else {
			http.Error(w, "unkown server error: "+err.Error(), http.StatusInternalServerError)
		}
		return
	}
	err = json.NewEncoder(w).Encode(api.MessageResponse{Msg: "avatar was deleted"})
	if err != nil {
		slog.Error("DeleteUserAvatar: can't encode json", "err", err)
	}
}
