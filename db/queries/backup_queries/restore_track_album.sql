-- name: RestoreTrackAlbum :exec
INSERT INTO public."track_album" (
    track_id, album_id
) VALUES (
    $1, $2
);
