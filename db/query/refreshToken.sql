
-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
    id,
    user_id,
    token_hash,
    user_agent,
    expires_at,
    client_ip
) VALUES (
    $1, $2, $3, $4 ,$5 , $6
)
RETURNING *;


-- name: GetRefreshTokenByHash :one
SELECT *
FROM refresh_tokens
WHERE token_hash = $1
LIMIT 1;


-- name: ListRefreshTokensByUserID :many
SELECT *
FROM refresh_tokens
WHERE user_id = $1
ORDER BY created_at DESC;


-- name: RevokeRefreshToken :one
UPDATE refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = $1
  AND revoked_at IS NULL
RETURNING *;


-- name: DeleteExpiredRefreshTokens :exec
DELETE FROM refresh_tokens
WHERE expires_at < NOW();


-- name: BlockRefreshToken :one
UPDATE refresh_tokens
SET is_blocked = $2
WHERE id = $1
RETURNING *;
