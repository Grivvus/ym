-- name: UpdateUserPasswordHashParams :exec
UPDATE "user" SET
    password_memory = $2,
    password_iterations = $3,
    password_parallelism = $4,
    password_key_length = $5,
    updated_at = now()
WHERE id = $1;
