-- +goose Up
-- +goose StatementBegin
CREATE TABLE playlist_share_info (
    playlist_id INTEGER NOT NULL,
    FOREIGN KEY (playlist_id) REFERENCES playlist (id) ON DELETE CASCADE,
    shared_with_user INTEGER NOT NULL,
    FOREIGN KEY(shared_with_user) REFERENCES "user"(id) ON DELETE CASCADE,
    PRIMARY KEY(playlist_id, shared_with_user),
    has_write_permission BOOLEAN NOT NULL DEFAULT false
);

CREATE INDEX playlist_share_info_user ON "playlist_share_info" (shared_with_user);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS playlist_share_info_user;
DROP TABLE IF EXISTS "playlist_share_info";
-- +goose StatementEnd
