-- +goose Up
-- +goose StatementBegin
ALTER TABLE "user"
    ADD COLUMN password_memory INTEGER NOT NULL DEFAULT 65536,
    ADD COLUMN password_iterations INTEGER NOT NULL DEFAULT 3,
    ADD COLUMN password_parallelism INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN password_key_length INTEGER NOT NULL DEFAULT 32,
    ADD CONSTRAINT user_password_memory_positive CHECK (password_memory > 0),
    ADD CONSTRAINT user_password_iterations_positive CHECK (password_iterations > 0),
    ADD CONSTRAINT user_password_parallelism_valid CHECK (
        password_parallelism >= 0 AND password_parallelism <= 255
    ),
    ADD CONSTRAINT user_password_key_length_positive CHECK (password_key_length > 0);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE "user"
    DROP CONSTRAINT IF EXISTS user_password_key_length_positive,
    DROP CONSTRAINT IF EXISTS user_password_parallelism_valid,
    DROP CONSTRAINT IF EXISTS user_password_iterations_positive,
    DROP CONSTRAINT IF EXISTS user_password_memory_positive,
    DROP COLUMN IF EXISTS password_key_length,
    DROP COLUMN IF EXISTS password_parallelism,
    DROP COLUMN IF EXISTS password_iterations,
    DROP COLUMN IF EXISTS password_memory;
-- +goose StatementEnd
