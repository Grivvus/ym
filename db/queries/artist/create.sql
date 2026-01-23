-- name: CreateArtist :one
INSERT INTO "artist" (name)
values ($1)
RETURNING *;
