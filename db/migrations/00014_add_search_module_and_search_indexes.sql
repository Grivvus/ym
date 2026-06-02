-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE INDEX track_name_trgm_idx ON track USING gin (name gin_trgm_ops);
CREATE INDEX album_name_trgm_idx ON album USING gin (name gin_trgm_ops);
CREATE INDEX artist_name_trgm_idx ON artist USING gin (name gin_trgm_ops);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS artist_name_trgm_idx;
DROP INDEX IF EXISTS album_name_trgm_idx;
DROP INDEX IF EXISTS track_name_trgm_idx;

DROP EXTENSION IF EXISTS pg_trgm;
-- +goose StatementEnd
