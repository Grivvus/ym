package handlers

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/Grivvus/ym/internal/api"
	"github.com/Grivvus/ym/internal/service"
)

type BackupHandlers struct {
	logger        *slog.Logger
	backupService service.BackupService
	authService   service.AuthService
}

func (h BackupHandlers) Backup(w http.ResponseWriter, r *http.Request, params api.BackupParams) {
	ok := requireSuperuser(w, r, h.authService)
	if !ok {
		return
	}
	backup, contentLen, err := h.backupService.MakeBackup(r.Context(), service.BackupSettings{
		IncludeImages:           params.IncludeImages != nil && *params.IncludeImages,
		IncludeTranscodedTracks: params.IncludeTranscodedTracks != nil && *params.IncludeTranscodedTracks,
	})
	defer func() { _ = backup.Close() }()
	if err != nil {
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	w.Header().Set("Content-Length", strconv.FormatInt(int64(contentLen), 10))
	w.Header().Set("Content-Type", "application/zip")
	_, err = io.Copy(w, backup)
	if err != nil {
		h.logger.Error("Failed to write response", "err", err)
	}
}

func (h BackupHandlers) Restore(w http.ResponseWriter, r *http.Request) {
	ok := requireSuperuser(w, r, h.authService)
	if !ok {
		return
	}
	id, err := h.backupService.StartRestoreOperation(r.Context(), r.Body)
	if err != nil {
		if _, ok := errors.AsType[service.ErrAlreadyExists](err); ok {
			_ = WriteError(w, http.StatusConflict, err)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
		return
	}
	_ = WriteJSON(w, http.StatusAccepted, id)
}

func (h BackupHandlers) GetRestoreStatus(
	w http.ResponseWriter, r *http.Request, restoreID string,
) {
	ok := requireSuperuser(w, r, h.authService)
	if !ok {
		return
	}
	resp, err := h.backupService.CheckRestoreOperation(r.Context(), restoreID)
	if err != nil {
		if _, ok := errors.AsType[service.ErrNotFound](err); ok {
			_ = WriteError(w, http.StatusNotFound, err)
		}
		if errors.Is(err, service.ErrBadParams) {
			_ = WriteError(w, http.StatusBadRequest, err)
			return
		}
		_ = WriteError(w, http.StatusInternalServerError, err)
	}
	_ = WriteJSON(w, http.StatusOK, resp)
}
