-- name: UpdatePasswordResetCodeAttempts :exec
UPDATE "password_reset_code" SET
    attempts_left = $2,
    updated_at = now()
WHERE user_id = $1;
