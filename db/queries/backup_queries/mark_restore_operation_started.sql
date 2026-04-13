-- name: MarkRestoreOperationStarted :exec
UPDATE public."restore_status" SET
    status = 'started'
WHERE id = $1;
