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

type AuthHandlers struct {
	service service.AuthService
}

func (h AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
	var user api.UserAuth
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Login(ctx, user)
	if err != nil {
		if err.Error() == "wrong password" {
			http.Error(w, "Invalid password", http.StatusBadRequest)
		} else if errors.Is(err, service.ErrNotFound{}) {
			http.Error(w, "Invalid username", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusOK)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("AuthHandlers.Login", "err", err)
	}
}

func (h AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	ctx := context.TODO()
	var user api.UserAuth
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Register(ctx, user)
	if err != nil {
		if errors.Is(err, service.ErrUserAlreadyExists{}) {
			http.Error(w, "User already exists", http.StatusBadRequest)
		} else {
			http.Error(w, "", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(resp)
	if err != nil {
		slog.Error("AuthHandlers.Register", "err", err)
	}
}
