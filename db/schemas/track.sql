CREATE TABLE track (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    duration INTEGER NOT NULL,
    is_uploaded_by_user BOOLEAN NOT NULL,
    url TEXT NOT NULL UNIQUE,
    artist_id INTEGER NOT NULL,
    album_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE,
    FOREIGN KEY (album_id) REFERENCES album (id) ON DELETE CASCADE
);
