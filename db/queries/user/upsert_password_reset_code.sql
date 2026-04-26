-- name: UpsertPasswordResetCode :exec
INSERT INTO "password_reset_code" (
    user_id, code_hash, expires_at, attempts_left,
    resend_available_at, created_at, updated_at
) VALUES (
    $1, $2, $3, $4,
    $5, now(), now()
)
ON CONFLICT (user_id) DO UPDATE SET
    code_hash = EXCLUDED.code_hash,
    expires_at = EXCLUDED.expires_at,
    attempts_left = EXCLUDED.attempts_left,
    resend_available_at = EXCLUDED.resend_available_at,
    created_at = EXCLUDED.created_at,
    updated_at = now();
