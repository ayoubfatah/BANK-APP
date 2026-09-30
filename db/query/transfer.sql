-- name: CreateTransfer :one
INSERT INTO transfers (
  from_account_id,
  to_account_id,
  amount,
  currency,
  status,
  reference,
  idempotency_key
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetTransfer :one
SELECT * FROM transfers
WHERE id = $1;

-- name: GetTransferForUpdate :one
SELECT * FROM transfers
WHERE id = $1
FOR NO KEY UPDATE;

-- name: GetTransferByIdempotencyKey :one
SELECT * FROM transfers
WHERE idempotency_key = $1;

-- name: ListTransfersByAccountID :many
SELECT * FROM transfers
WHERE from_account_id = $1 OR to_account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransfersByUserID :many
SELECT t.* FROM transfers t
JOIN accounts a ON a.id = t.from_account_id OR a.id = t.to_account_id
WHERE a.user_id = $1
ORDER BY t.created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateTransferStatus :one
UPDATE transfers
SET status = $2::varchar,
    failure_reason = $3,
    completed_at = CASE WHEN $2::varchar = 'completed' THEN now() ELSE completed_at END
WHERE id = $1
RETURNING *;