-- name: CreateUser :one
INSERT INTO "user" (username, email, password, created_at, updated_at)
values ($1, $2, $3, now(), now())
RETURNING *;
