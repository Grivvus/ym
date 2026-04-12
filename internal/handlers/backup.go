package handlers

import (
	"log/slog"
	"net/http"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type BackupHandlers struct {
	logger        *slog.Logger
	backupService service.BackupService
}

func (h BackupHandlers) Backup(w http.ResponseWriter, r *http.Request, params api.BackupParams) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h BackupHandlers) Restore(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotImplemented)
}

func (h BackupHandlers) GetRestoreStatus(
	w http.ResponseWriter, r *http.Request, restoreID string,
) {
	w.WriteHeader(http.StatusNotImplemented)
}
