-- name: DeleteFromTranscodingQueue :one
DELETE FROM public."transcoding_queue"
    WHERE id = $1
    RETURNING *;