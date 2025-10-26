-- +goose Up
-- +goose StatementBegin
CREATE TABLE album (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "album"
-- +goose StatementEnd
