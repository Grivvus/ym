-- name: DeleteTrackFromPlaylist :exec
DELETE FROM "track_playlist"
    WHERE track_id = $1;

-- name: DeleteTrackFromPlaylistRelation :exec
DELETE FROM "track_playlist"
    WHERE playlist_id = $1
        AND track_id = $2;
