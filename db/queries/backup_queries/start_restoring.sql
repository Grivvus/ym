-- name: StartRestoreOperation :one
INSERT INTO public."restore_status" (id, status)
    VALUES ($1, 'pending')
    RETURNING id;
