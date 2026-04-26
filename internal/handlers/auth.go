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
	authService          service.AuthService
	passwordResetService service.PasswordResetService
	logger               *slog.Logger
}

func (h AuthHandlers) Login(w http.ResponseWriter, r *http.Request) {
	var user api.UserAuth
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.authService.Login(r.Context(), user)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			_ = WriteError(
				w, http.StatusUnauthorized, fmt.Errorf("invalid credentials"),
			)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(
				w, http.StatusUnauthorized, fmt.Errorf("invalid credentials"),
			)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var user api.UserAuth
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.authService.Register(r.Context(), user)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = WriteError(w, http.StatusConflict, fmt.Errorf("user already exists"))
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = WriteJSON(w, http.StatusCreated, resp)
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req api.UpdateTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	resp, err := h.authService.UpdateTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			_ = WriteError(w, http.StatusUnauthorized, fmt.Errorf("invalid refresh token"))
		} else {
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req api.PasswordResetRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	err = h.passwordResetService.RequestPasswordReset(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrServiceUnavailable):
			_ = WriteError(w, http.StatusServiceUnavailable, err)
		case errors.Is(err, service.ErrBadParams):
			_ = WriteError(w, http.StatusBadRequest, err)
		default:
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = WriteJSON(w, http.StatusAccepted, api.MessageResponse{
		Msg: h.passwordResetService.AcceptedMessage(),
	})
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}

func (h AuthHandlers) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req api.PasswordResetConfirmRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		_ = WriteError(
			w, http.StatusBadRequest, fmt.Errorf("invalid body: %w", err),
		)
		return
	}

	err = h.passwordResetService.ConfirmPasswordReset(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrServiceUnavailable):
			_ = WriteError(w, http.StatusServiceUnavailable, err)
		case errors.Is(err, service.ErrBadParams):
			_ = WriteError(w, http.StatusBadRequest, err)
		default:
			_ = WriteError(w, http.StatusInternalServerError, err)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, api.MessageResponse{
		Msg: "password was successfully reset",
	})
	if err != nil {
		h.logger.Error("failed to encode response", "err", err)
	}
}
