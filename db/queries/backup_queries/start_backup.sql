-- name: StartBackupOperation :one
INSERT INTO public."backup_status" (
    id, status, include_images, include_transcoded_tracks
) VALUES (
    $1, 'pending', $2, $3
)
RETURNING id;
