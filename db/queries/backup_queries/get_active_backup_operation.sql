-- name: GetActiveBackupOperation :one
SELECT id, status, error, created_at, finished_at,
    include_images, include_transcoded_tracks, archive_path, size_bytes
FROM public."backup_status"
WHERE status IN ('pending', 'started')
ORDER BY created_at DESC
LIMIT 1;
