-- +goose Up
-- +goose StatementBegin
CREATE TABLE track (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    duration INTEGER,
    url TEXT UNIQUE,
    fast_preset_fname TEXT,
    standard_preset_fname TEXT,
    high_preset_fname TEXT,
    lossless_preset_fname TEXT,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "track"
-- +goose StatementEnd
