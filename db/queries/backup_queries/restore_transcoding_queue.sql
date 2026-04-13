-- name: RestoreTranscodingQueue :exec
INSERT INTO public."transcoding_queue" (
    id, added_at, track_original_file_name,
    track_id, was_failed, error_msg
) VALUES (
    $1, $2, $3,
    $4, $5, $6
);
