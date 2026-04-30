-- +goose Up
-- +goose StatementBegin
ALTER TABLE public."user"
    ADD COLUMN refresh_version INTEGER NOT NULL DEFAULT 0;

CREATE TABLE public."password_reset_code" (
    user_id INTEGER PRIMARY KEY REFERENCES public."user" (id) ON DELETE CASCADE,
    code_hash bytea NOT NULL,
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    attempts_left INTEGER NOT NULL,
    resend_available_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS public."password_reset_code";

ALTER TABLE public."user"
    DROP COLUMN IF EXISTS refresh_version;
-- +goose StatementEnd
