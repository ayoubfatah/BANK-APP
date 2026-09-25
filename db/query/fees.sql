-- name: CreateFee :one
INSERT INTO fees (
    transfer_id,
    type,
    amount,
    currency
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING *;


-- name: GetFee :one
SELECT *
FROM fees
WHERE id = $1
LIMIT 1;


-- name: ListFeesByTransferID :many
SELECT *
FROM fees
WHERE transfer_id = $1
ORDER BY created_at DESC;


-- name: GetTransferFees :many
SELECT *
FROM fees
WHERE transfer_id = $1
ORDER BY created_at DESC;