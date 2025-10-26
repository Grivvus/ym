-- +goose Up
-- +goose StatementBegin
CREATE TABLE artist (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "artist"
-- +goose StatementEnd
