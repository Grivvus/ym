-- name: UpdateUserPassword :exec
UPDATE "user" SET 
    password = $2,
    salt = $3,
    updated_at = now()
WHERE id = $1;
