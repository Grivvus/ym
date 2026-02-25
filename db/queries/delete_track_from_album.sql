-- name: DeleteTrackFromAlbum :exec
DELETE FROM "track_album"
    WHERE track_id = $1;
