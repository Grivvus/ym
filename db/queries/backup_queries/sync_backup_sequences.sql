-- name: SyncUserSequence :one
SELECT setval(
    pg_get_serial_sequence('public."user"', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."user";

-- name: SyncArtistSequence :one
SELECT setval(
    pg_get_serial_sequence('public.artist', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."artist";

-- name: SyncAlbumSequence :one
SELECT setval(
    pg_get_serial_sequence('public.album', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."album";

-- name: SyncPlaylistSequence :one
SELECT setval(
    pg_get_serial_sequence('public.playlist', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."playlist";

-- name: SyncTrackSequence :one
SELECT setval(
    pg_get_serial_sequence('public.track', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."track";

-- name: SyncTranscodingQueueSequence :one
SELECT setval(
    pg_get_serial_sequence('public.transcoding_queue', 'id'),
    GREATEST(COALESCE(MAX(id), 0), 1),
    COALESCE(MAX(id), 0) > 0
)
FROM public."transcoding_queue";
