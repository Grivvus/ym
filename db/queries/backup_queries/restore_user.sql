-- name: RestoreUser :exec
INSERT INTO public."user" (
    id, username, is_superuser, email,
    password, salt, refresh_version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, $6, $7, $8, $9
);
