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
