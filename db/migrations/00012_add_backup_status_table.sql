-- +goose Up
-- +goose StatementBegin
CREATE TABLE backup_status (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    finished_at TIMESTAMP WITH TIME ZONE,
    status status NOT NULL DEFAULT 'pending',
    error TEXT,
    include_images BOOLEAN NOT NULL,
    include_transcoded_tracks BOOLEAN NOT NULL,
    archive_path TEXT,
    size_bytes BIGINT
);

CREATE INDEX backup_status_active_created_at_idx
    ON "backup_status" (created_at DESC)
    WHERE status IN ('pending', 'started');
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS backup_status_active_created_at_idx;
DROP TABLE IF EXISTS "backup_status";
-- +goose StatementEnd
