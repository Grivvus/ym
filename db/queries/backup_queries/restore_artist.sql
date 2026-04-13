-- name: RestoreArtist :exec
INSERT INTO public."artist" (
    id, name, url
) VALUES (
    $1, $2, $3
);
