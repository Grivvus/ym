-- name: DeleteTrack :exec
DELETE from "track"
    where id = $1;
