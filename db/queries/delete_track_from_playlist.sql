-- name: DeleteTrackFromPlaylist :exec
DELETE FROM "track_playlist"
    WHERE track_id = $1;
