-- name: GetRestoreStatus :one
SELECT id, status, error, created_at, finished_at
FROM public."restore_status"
    WHERE id = $1;