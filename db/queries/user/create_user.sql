-- name: CreateUser :one
INSERT INTO "user" (
    username, email, password, salt,
    password_memory, password_iterations, password_parallelism, password_key_length,
    is_superuser, created_at, updated_at
)
values (
    $1, $2, $3, $4,
    $5, $6, $7, $8,
    $9, now(), now()
)
RETURNING *;
