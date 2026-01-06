package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

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
		if errors.Is(err, service.ErrNoSuchUser{}) {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}

	w.WriteHeader(http.StatusOK)
	b, err := json.Marshal(user)
	if err != nil {
		slog.Error("Error while marshalling model", "err", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	_, err = w.Write(b)
	if err != nil {
		slog.Error("Unexpected error", "err", err)
	}
}

func (u UserHandlers) ChangeUser(w http.ResponseWriter, r *http.Request, userId int) {}

func (u UserHandlers) ChangePassword(w http.ResponseWriter, r *http.Request, userId int) {}

func (u UserHandlers) UploadUserAvatar(w http.ResponseWriter, r *http.Request, userId int) {}
