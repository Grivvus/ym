CREATE TABLE track (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    duration INTEGER,
    fast_preset_fname TEXT,
    standard_preset_fname TEXT,
    high_preset_fname TEXT,
    lossless_preset_fname TEXT,
    artist_id INTEGER NOT NULL,
    FOREIGN KEY (artist_id) REFERENCES artist (id) ON DELETE CASCADE
);
