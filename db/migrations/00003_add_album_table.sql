-- +goose Up
-- +goose StatementBegin
CREATE TABLE album (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    release_year INTEGER,
    release_full_date DATE,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE,
    UNIQUE(name, artist_id)
);

CREATE INDEX album_release_year ON "album" (release_year)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "album"
-- +goose StatementEnd
