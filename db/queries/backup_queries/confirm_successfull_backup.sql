-- name: ConfirmBackup :exec
UPDATE public."backup_status" SET
    status = 'finished',
    error = NULL,
    finished_at = now(),
    archive_path = $2,
    size_bytes = $3
WHERE id = $1;
