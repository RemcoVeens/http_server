-- name: GenerateToken :one
INSERT INTO refresh_token (token, created_at, updated_at, user_id, expires_at, revoked_at)
VALUES (
$1, NOW(),NOW(), $2, $3,$4
)
returning *;

-- name: GetTokenFromToken :one
SELECT * FROM refresh_token WHERE token = $1;

-- name: RevokeToken :exec
UPDATE refresh_token
SET revoked_at= NOW(), updated_at= NOW()
WHERE token =$1;

-- name: GetUserFromRefreshToken :one
SELECT u.* FROM users u
INNER JOIN refresh_token rt ON rt.user_id = u.id
WHERE rt.token = $1;
