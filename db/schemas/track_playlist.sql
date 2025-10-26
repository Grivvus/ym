CREATE TABLE track_playlist (
    track_id INTEGER NOT NULL,
    playlist_id INTEGER NOT NULL,
    PRIMARY KEY (track_id, playlist_id),
    FOREIGN KEY (track_id) REFERENCES track (id) ON DELETE CASCADE,
    FOREIGN KEY (playlist_id) REFERENCES playlist (id) ON DELETE CASCADE
);
