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
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}

	resp, err := h.authService.Login(r.Context(), user)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(
				w, http.StatusUnauthorized,
				fmt.Errorf("invalid credentials"), h.logger,
			)
			return
		}
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			MustWriteError(
				w, http.StatusUnauthorized,
				fmt.Errorf("invalid credentials"), h.logger,
			)
			return
		}
		MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AuthHandlers) Register(w http.ResponseWriter, r *http.Request) {
	var user api.UserAuth
	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}

	resp, err := h.authService.Register(r.Context(), user)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			MustWriteError(
				w, http.StatusConflict,
				fmt.Errorf("user already exists"), h.logger,
			)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		}
		return
	}

	err = WriteJSON(w, http.StatusCreated, resp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AuthHandlers) RefreshTokens(w http.ResponseWriter, r *http.Request) {
	var req api.UpdateTokenRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}

	resp, err := h.authService.UpdateTokens(r.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, service.ErrUnauthorized) {
			MustWriteError(
				w, http.StatusUnauthorized,
				fmt.Errorf("invalid refresh token"), h.logger,
			)
		} else {
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, resp)
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AuthHandlers) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req api.PasswordResetRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}

	err = h.passwordResetService.RequestPasswordReset(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrServiceUnavailable):
			MustWriteError(w, http.StatusServiceUnavailable, err, h.logger)
		case errors.Is(err, service.ErrBadParams):
			MustWriteError(w, http.StatusBadRequest, err, h.logger)
		default:
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		}
		return
	}

	err = WriteJSON(w, http.StatusAccepted, api.MessageResponse{
		Msg: h.passwordResetService.AcceptedMessage(),
	})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}

func (h AuthHandlers) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req api.PasswordResetConfirmRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		MustWriteError(
			w, http.StatusBadRequest,
			fmt.Errorf("invalid body: %w", err), h.logger,
		)
		return
	}

	err = h.passwordResetService.ConfirmPasswordReset(r.Context(), req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrServiceUnavailable):
			MustWriteError(w, http.StatusServiceUnavailable, err, h.logger)
		case errors.Is(err, service.ErrBadParams):
			MustWriteError(w, http.StatusBadRequest, err, h.logger)
		default:
			MustWriteError(w, http.StatusInternalServerError, err, h.logger)
		}
		return
	}

	err = WriteJSON(w, http.StatusOK, api.MessageResponse{
		Msg: "password was successfully reset",
	})
	if err != nil {
		h.logger.Error("can't encode response", "err", err)
	}
}
