-- name: RestoreTrackPlaylist :exec
INSERT INTO public."track_playlist" (
    track_id, playlist_id
) VALUES (
    $1, $2
);
