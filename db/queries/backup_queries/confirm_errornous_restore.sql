-- name: ErrorRestoring :exec
UPDATE public."restore_status" SET
    status = 'error',
    error = $2,
    finished_at = now()
WHERE id = $1;
