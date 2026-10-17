-- name: CreateConversion :one
INSERT INTO conversions (
  user_id,
  from_account_id,
  to_account_id,
  from_amount,
  to_amount,
  from_currency,
  to_currency,
  exchange_rate,
  fee,
  status,
  idempotency_key
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
)
RETURNING *;

-- name: GetConversionByID :one
SELECT * FROM conversions
WHERE id = $1;

-- name: GetConversionForUpdate :one
SELECT * FROM conversions
WHERE id = $1
FOR NO KEY UPDATE;

-- name: GetConversionByIdempotencyKey :one
SELECT * FROM conversions
WHERE idempotency_key = $1;

-- name: ListConversionsByUserID :many
SELECT * FROM conversions
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;



-- name: ListConversionsAdmin :many
SELECT *
FROM conversions
WHERE (sqlc.narg('user_id')::bigint        IS NULL OR user_id       = sqlc.narg('user_id'))
  AND (sqlc.narg('from_currency')::text    IS NULL OR from_currency = sqlc.narg('from_currency'))
  AND (sqlc.narg('to_currency')::text      IS NULL OR to_currency   = sqlc.narg('to_currency'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at   >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz   IS NULL OR created_at    < sqlc.narg('to_date'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');
 
 
-- name: UpdateConversionStatus :one
UPDATE conversions
SET status = sqlc.arg('status')
WHERE id = sqlc.arg('id')
RETURNING *;
 