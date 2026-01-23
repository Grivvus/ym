-- name: DeleteAlbum :exec
DELETE from "album"
    where id = $1;
