-- name: GetPasswordResetCodeByUserID :one
SELECT user_id, code_hash, expires_at, attempts_left, resend_available_at, created_at, updated_at
FROM "password_reset_code"
WHERE user_id = $1;
