-- name: DeleteArtist :exec
DELETE FROM "artist" 
    where id = $1;
