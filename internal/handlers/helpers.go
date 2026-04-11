package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)

func WriteJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, err error) error {
	return WriteJSON(w, status, api.ErrorResponse{
		Error: err.Error(),
	})
}

func FormValueToBool(val string) bool {
	return val == "true" || val == "True"
}
