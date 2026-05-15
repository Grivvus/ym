CREATE TABLE backup_status (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT now(),
    finished_at TIMESTAMP WITH TIME ZONE,
    status status NOT NULL DEFAULT 'pending',
    error TEXT,
    include_images BOOLEAN NOT NULL,
    include_transcoded_tracks BOOLEAN NOT NULL,
    archive_path TEXT,
    size_bytes BIGINT
);
