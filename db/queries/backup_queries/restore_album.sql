-- name: RestoreAlbum :exec
INSERT INTO public."album" (
    id, name, artist_id, release_year, release_full_date
) VALUES (
    $1, $2, $3, $4, $5
);
