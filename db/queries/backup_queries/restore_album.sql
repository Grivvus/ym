-- name: RestoreAlbum :exec
INSERT INTO public."album" (
    id, name, artist_id
) VALUES (
    $1, $2, $3
);
