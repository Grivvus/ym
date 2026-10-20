-- name: TranscodingQueueSize :one
select COUNT(*) from public."transcoding_queue";

-- name: TranscodingQueueFailed :one
select COUNT(*) from public."transcoding_queue"
    where was_failed = true;
