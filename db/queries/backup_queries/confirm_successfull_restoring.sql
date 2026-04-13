-- name: ConfirmRestoring :exec
UPDATE public."restore_status" SET
    status = 'finished',
    error = NULL,
    finished_at = now()
WHERE id = $1;
