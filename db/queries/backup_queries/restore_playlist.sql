-- name: RestorePlaylist :exec
INSERT INTO public."playlist" (
    id, name, is_public, owner_id
) VALUES (
    $1, $2, $3, $4
);
