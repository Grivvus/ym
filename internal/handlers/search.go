package handlers

import (
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type SearchHandler struct {
	logger  *slog.Logger
	service service.SearchService
}

func (h SearchHandler) Search(w http.ResponseWriter, r *http.Request, params api.SearchParams) {
	WriteError(w, http.StatusNotImplemented, fmt.Errorf("not implemented"))
}
