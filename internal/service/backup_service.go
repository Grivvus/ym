package service

import (
	"log/slog"
)

type BackupService struct {
	logger *slog.Logger
}

func NewBackupService(logger *slog.Logger) BackupService {
	return BackupService{logger: logger}
}
