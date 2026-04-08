-- name: CreateUser :one
INSERT INTO "user" (username, email, password, salt, is_superuser, created_at, updated_at)
values ($1, $2, $3, $4, $5, now(), now())
RETURNING *;
