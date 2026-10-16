-- name: CreateFeeRule :one
INSERT INTO fee_rules (
    operation_type,
    currency,
    fee_type,
    flat_amount,
    percentage_bps,
    min_fee,
    max_fee
) VALUES (
    sqlc.arg('operation_type'),
    sqlc.arg('currency'),
    sqlc.arg('fee_type'),
    sqlc.arg('flat_amount'),
    sqlc.arg('percentage_bps'),
    sqlc.arg('min_fee'),
    sqlc.narg('max_fee')
)
RETURNING *;


-- name: GetFeeRule :one
SELECT *
FROM fee_rules
WHERE id = $1
LIMIT 1;


-- name: GetActiveFeeRule :one
SELECT *
FROM fee_rules
WHERE operation_type = $1
  AND currency = $2
  AND is_active
LIMIT 1;


-- name: ListActiveFeeRules :many
SELECT *
FROM fee_rules
WHERE is_active
ORDER BY operation_type, currency;


-- name: ListFeeRules :many
SELECT *
FROM fee_rules
WHERE (sqlc.narg('operation_type')::text IS NULL OR operation_type = sqlc.narg('operation_type'))
  AND (sqlc.narg('currency')::text       IS NULL OR currency       = sqlc.narg('currency'))
  AND (sqlc.narg('is_active')::boolean   IS NULL OR is_active      = sqlc.narg('is_active'))
ORDER BY operation_type, currency, id;


-- name: UpdateFeeRule :one
UPDATE fee_rules
SET
    fee_type       = COALESCE(sqlc.narg('fee_type'), fee_type),
    flat_amount    = COALESCE(sqlc.narg('flat_amount'), flat_amount),
    percentage_bps = COALESCE(sqlc.narg('percentage_bps'), percentage_bps),
    min_fee        = COALESCE(sqlc.narg('min_fee'), min_fee),
    max_fee        = COALESCE(sqlc.narg('max_fee'), max_fee),
    is_active      = COALESCE(sqlc.narg('is_active'), is_active),
    updated_at     = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;


-- name: ListFeesAdmin :many
SELECT *
FROM fees
WHERE (sqlc.narg('fee_type')::text         IS NULL OR type     = sqlc.narg('fee_type'))
  AND (sqlc.narg('currency')::text         IS NULL OR currency = sqlc.narg('currency'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz   IS NULL OR created_at  < sqlc.narg('to_date'))
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');


-- name: SumFeesAdmin :many
SELECT
    type,
    currency,
    COUNT(*)::bigint                 AS fee_count,
    COALESCE(SUM(amount), 0)::bigint AS total_amount
FROM fees
WHERE (sqlc.narg('fee_type')::text         IS NULL OR type     = sqlc.narg('fee_type'))
  AND (sqlc.narg('currency')::text         IS NULL OR currency = sqlc.narg('currency'))
  AND (sqlc.narg('from_date')::timestamptz IS NULL OR created_at >= sqlc.narg('from_date'))
  AND (sqlc.narg('to_date')::timestamptz   IS NULL OR created_at  < sqlc.narg('to_date'))
GROUP BY type, currency
ORDER BY type, currency;