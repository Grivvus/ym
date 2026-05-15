-- name: ErrorBackup :exec
UPDATE public."backup_status" SET
    status = 'error',
    error = $2,
    finished_at = now()
WHERE id = $1;
