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
    is_globally_available BOOLEAN NOT NULL,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE,
    upload_by_user INTEGER,
    FOREIGN KEY (upload_by_user) REFERENCES "user" (id) ON DELETE SET NULL
);

CREATE INDEX track_fk_on_artist ON "track" (artist_id);
CREATE INDEX track_fk_on_user ON "track" (upload_by_user);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE "track"
DROP INDEX IF EXISTS track_fk_on_user;
DROP INDEX IF EXISTS track_fk_on_artist;
-- +goose StatementEnd
