-- name: ErrorRestoring :exec
UPDATE public."restore_status" SET
    status = 'error',
    error = $1
;