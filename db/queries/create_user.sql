-- name: CreateUser :one
INSERT INTO "user" (username, email, password, salt, created_at, updated_at)
values ($1, $2, $3, $4, now(), now())
RETURNING *;
