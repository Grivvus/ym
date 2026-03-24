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

type AuthHandlers struct {
	service service.AuthService
	logger  *slog.Logger
}

func (h AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var user api.UserAuth
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.service.Login(r.Context(), user)
	if err != nil {
		if err.Error() == "wrong password" {
			_ = writeError(
				w, http.StatusBadRequest, fmt.Errorf("invalid password"),
			)
		} else if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(
				w, http.StatusBadRequest, fmt.Errorf("invalid username"),
			)
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = writeJSON(w, http.StatusOK, resp)
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var user api.UserAuth
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.service.Register(r.Context(), user)
	if err != nil {
		if _, ok := errors.AsType[service.ErrUserAlreadyExists](err); ok {
			_ = writeError(w, http.StatusBadRequest, fmt.Errorf("user already exists"))
		} else {
			_ = writeError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = writeJSON(w, http.StatusCreated, resp)
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	_ = writeError(w, http.StatusNotImplemented, fmt.Errorf("not implemented"))
}
