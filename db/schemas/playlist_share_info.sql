CREATE TABLE playlist_share_info (
    playlist_id INTEGER NOT NULL,
    FOREIGN KEY (playlist_id) REFERENCES playlist (id) ON DELETE CASCADE,
    shared_with_user INTEGER NOT NULL,
    FOREIGN KEY(shared_with_user) REFERENCES "user"(id) ON DELETE CASCADE,
    PRIMARY KEY(playlist_id, shared_with_user),
    has_write_permission BOOLEAN NOT NULL DEFAULT false
);
