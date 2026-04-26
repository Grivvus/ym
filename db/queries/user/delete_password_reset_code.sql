-- name: DeletePasswordResetCodeByUserID :exec
DELETE FROM "password_reset_code"
WHERE user_id = $1;
