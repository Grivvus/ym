-- +goose Up
-- +goose StatementBegin
CREATE TABLE track (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    duration INTEGER NOT NULL,
    url TEXT UNIQUE,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "track"
-- +goose StatementEnd
