-- +goose Up
-- +goose StatementBegin
CREATE TYPE status as ENUM ('pending', 'started', 'finished', 'error');
CREATE TABLE restore_status(
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    finished_at TIMESTAMP WITH TIME ZONE,
    status status NOT NULL DEFAULT 'pending',
    error TEXT
);

CREATE INDEX restore_status_active_created_at_idx
    ON "restore_status" (created_at DESC)
    WHERE status IN ('pending', 'started');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS restore_status_active_created_at_idx;
DROP TABLE IF EXISTS "restore_status";
DROP TYPE status
-- +goose StatementEnd
