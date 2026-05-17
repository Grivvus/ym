-- name: DeleteTrackFromAlbum :exec
DELETE FROM "track_album"
    WHERE track_id = $1;

-- name: DeleteTrackFromAlbumRelation :exec
DELETE FROM "track_album"
    WHERE album_id = $1
        AND track_id = $2;
