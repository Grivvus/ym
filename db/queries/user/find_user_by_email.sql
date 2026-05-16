-- name: GetUserByEmail :one
SELECT id, username, email, password, salt,
    password_memory, password_iterations, password_parallelism, password_key_length,
    is_superuser, refresh_version
FROM "user"
WHERE LOWER(email) = LOWER($1);
