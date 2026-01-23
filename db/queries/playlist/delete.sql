-- name: DeletePlaylist :exec
DELETE from "playlist"
    where id = $1;
