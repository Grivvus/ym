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
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "restore_status"
DROP TYPE status
-- +goose StatementEnd
