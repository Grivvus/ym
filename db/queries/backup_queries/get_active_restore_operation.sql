-- name: GetActiveRestoreOperation :one
SELECT id, status, error, created_at, finished_at
FROM public."restore_status"
WHERE status IN ('pending', 'started')
ORDER BY created_at DESC
LIMIT 1;
