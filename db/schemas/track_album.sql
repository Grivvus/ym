CREATE TABLE track_album (
    track_id INTEGER NOT NULL,
    album_id INTEGER NOT NULL,
    PRIMARY KEY (track_id, album_id),
    FOREIGN KEY (track_id) REFERENCES track (id) ON DELETE CASCADE,
    FOREIGN KEY (playlist_id) REFERENCES album (id) ON DELETE CASCADE
);
