-- name: GetArtistsWithFilter :many
SELECT "id", "name" FROM artist
                    -- starts with
    WHERE "name" LIKE $1 || '%'
LIMIT $2;
