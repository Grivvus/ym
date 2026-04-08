-- name: GetTranscodingQueue :many
SELECT id, added_at, track_original_file_name, track_id
    FROM public."transcoding_queue"
    WHERE id > $1
    ORDER BY id ASC
    LIMIT $2;