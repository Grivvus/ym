-- +goose Up
-- +goose StatementBegin
CREATE TABLE track_album(
    track_id INTEGER NOT NULL,
    album_id INTEGER NOT NULL,
    PRIMARY KEY (track_id, album_id),
    FOREIGN KEY (track_id) REFERENCES track (id) ON DELETE CASCADE,
    FOREIGN KEY (album_id) REFERENCES album (id) ON DELETE CASCADE
);

CREATE INDEX track_album_album_id_first
    ON track_album (album_id, track_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS track_album_album_id_first;
DROP TABLE IF EXISTS "track_album";
-- +goose StatementEnd
