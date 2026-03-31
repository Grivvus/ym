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
	err := json.NewDecoder(r.Body).Decode(&user)
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
				w, http.StatusUnauthorized, fmt.Errorf("invalid credentials"),
			)
		} else if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = writeError(
				w, http.StatusUnauthorized, fmt.Errorf("invalid credentials"),
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
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.service.Register(r.Context(), user)
	if err != nil {
		if _, ok := errors.AsType[service.ErrUserAlreadyExists](err); ok {
			_ = writeError(w, http.StatusConflict, fmt.Errorf("user already exists"))
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
	var req api.UpdateTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = writeError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.service.UpdateTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			_ = writeError(w, http.StatusUnauthorized, fmt.Errorf("invalid refresh token"))
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
