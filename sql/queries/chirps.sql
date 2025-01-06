-- name: CreateChirp :one
INSERT INTO chirps(id, user_id, created_at, updated_at, body)
VALUES (
	$1, $2, NOW(), NOW(), $3
)
RETURNING *;

-- name: GetChirps :many
SELECT * 
FROM chirps
WHERE ($1 = '00000000-0000-0000-0000-000000000000'::UUID OR user_id = $1)
ORDER BY 
    CASE 
        WHEN $2 = 'desc' THEN created_at 
        ELSE NULL
    END DESC,
    CASE 
        WHEN $2 = 'asc' THEN created_at 
        ELSE NULL
    END ASC;

-- name: GetChirp :one
SELECT * FROM chirps 
WHERE id = $1;

-- name: DeleteChirp :exec
DELETE FROM chirps
WHERE id = $1;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;
