package handlers

import (
	"net/http"

	"github.com/Grivvus/ym/internal/api"
)
import (
	"encoding/json"
)

func writeJSON(w http.ResponseWriter, status int, v any) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) error {
	return writeJSON(w, status, api.ErrorResponse{
		Error: err.Error(),
	})
}
