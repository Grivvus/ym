package handlers

import (
	"encoding/json/v2"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.MarshalWrite(w, v)
}

func WriteError(w http.ResponseWriter, status int, err error) error {
	return WriteJSON(w, status, api.ErrorResponse{
		Error: err.Error(),
	})
}

func MustWriteError(w http.ResponseWriter, status int, err error, logger *slog.Logger) {
	errSend := WriteError(w, status, err)
	if errSend != nil {
		logger.Error("can't encode response", "err", errSend)
	}
}

func FormValueToBool(val string) bool {
	return val == "true" || val == "True"
}
