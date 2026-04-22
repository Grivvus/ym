CREATE TABLE transcoding_queue(
    id BIGSERIAL PRIMARY KEY,
    added_at timestamp NOT NULL DEFAULT now(),
    track_original_file_name TEXT NOT NULL,
    track_id INTEGER NOT NULL,
    FOREIGN KEY (track_id) REFERENCES "track" (id) ON DELETE CASCADE,
    was_failed BOOLEAN NOT NULL DEFAULT FALSE,
    error_msg TEXT
);
