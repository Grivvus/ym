-- name: UpdateUserPassword :exec
WITH updated_user AS (
    UPDATE "user" SET
        password = $2,
        salt = $3,
        password_memory = $4,
        password_iterations = $5,
        password_parallelism = $6,
        password_key_length = $7,
        refresh_version = refresh_version + 1,
        updated_at = now()
    WHERE id = $1
    RETURNING id
)
DELETE FROM "password_reset_code"
WHERE user_id IN (SELECT id FROM updated_user);
