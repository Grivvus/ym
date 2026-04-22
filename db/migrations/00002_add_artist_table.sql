-- +goose Up
-- +goose StatementBegin
CREATE TABLE artist (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    url TEXT
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS "artist";
-- +goose StatementEnd
