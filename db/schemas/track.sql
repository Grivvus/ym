CREATE TABLE track (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    duration INTEGER NOT NULL,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE,
);
