-- name: SyncBackupSequences :one
WITH
    user_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."user"
    ),
    artist_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."artist"
    ),
    album_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."album"
    ),
    playlist_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."playlist"
    ),
    track_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."track"
    ),
    transcoding_queue_max AS (
        SELECT COALESCE(MAX(id), 0) AS value
        FROM public."transcoding_queue"
    ),
    user_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public."user"', 'id'),
            GREATEST((SELECT value FROM user_max), 1),
            (SELECT value FROM user_max) > 0
        )
    ),
    artist_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public.artist', 'id'),
            GREATEST((SELECT value FROM artist_max), 1),
            (SELECT value FROM artist_max) > 0
        )
    ),
    album_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public.album', 'id'),
            GREATEST((SELECT value FROM album_max), 1),
            (SELECT value FROM album_max) > 0
        )
    ),
    playlist_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public.playlist', 'id'),
            GREATEST((SELECT value FROM playlist_max), 1),
            (SELECT value FROM playlist_max) > 0
        )
    ),
    track_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public.track', 'id'),
            GREATEST((SELECT value FROM track_max), 1),
            (SELECT value FROM track_max) > 0
        )
    ),
    transcoding_queue_seq AS (
        SELECT setval(
            pg_get_serial_sequence('public.transcoding_queue', 'id'),
            GREATEST((SELECT value FROM transcoding_queue_max), 1),
            (SELECT value FROM transcoding_queue_max) > 0
        )
    )
SELECT 1;
