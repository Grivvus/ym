-- name: ClearBackupData :exec
TRUNCATE TABLE
    public."track_playlist",
    public."track_album",
    public."transcoding_queue",
    public."track",
    public."playlist",
    public."album",
    public."artist",
    public."user"
RESTART IDENTITY CASCADE;
