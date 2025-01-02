-- name: CreateChirp :one
INSERT INTO chirps(id, user_id, created_at, updated_at, body)
VALUES (
	$1, $2, NOW(), NOW(), $3
)
RETURNING *;

-- name: DeleteAllChirps :exec
DELETE FROM chirps;
