-- name: MarkBackupOperationStarted :exec
UPDATE public."backup_status" SET
    status = 'started'
WHERE id = $1;
