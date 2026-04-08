-- name: UpdateToSuperuser :one
UPDATE public."user" SET
    is_superuser = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;
