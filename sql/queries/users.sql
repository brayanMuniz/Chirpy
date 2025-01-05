-- name: CreateUser :one
INSERT INTO users (id, created_at, updated_at, email, hashed_password)
VALUES (
	$1, NOW(), NOW(), $2, $3
)
RETURNING *;

-- name: UpdateUser :one
UPDATE users
SET email = $2, hashed_password = $3
WHERE id = $1
RETURNING *;

-- name: UpgradeToChirpyRed :exec
UPDATE users
SET is_chirpy_red = TRUE
WHERE id= $1;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: DeleteAll :exec
DELETE FROM users;
