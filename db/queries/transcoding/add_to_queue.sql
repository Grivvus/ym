-- name: AddToTranscodingQueue :one
INSERT INTO public."transcoding_queue" (track_original_file_name, track_id, was_failed, error_msg)
    VALUES ($1, $2, $3, $4)
    RETURNING *;