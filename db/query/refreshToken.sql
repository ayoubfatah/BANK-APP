-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (
  user_id ,
  token_hash,
  expires_at 
) VALUES (
  $1, $2, $3
)
RETURNING *;


-- name: GetRefreshTokenByHash :one
SELECT * FROM refresh_tokens
WHERE token_hash = $1 LIMIT 1;


-- name: ListRefreshTokensByUserID :many
SELECT * FROM refresh_tokens 
WHERE user_id = $1 
ORDER BY created_at DESC;


-- name: RevokeRefreshToken :one
UPDATE  refresh_tokens
SET revoked_at = NOW()
WHERE token_hash = $1
RETURNING * ;

-- name: DeleteExpiredRefreshTokens :exec
DELETE  FROM refresh_tokens 
WHERE  expires_at < NOW();