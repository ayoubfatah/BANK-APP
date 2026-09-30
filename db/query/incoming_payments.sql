-- name: CreateIncomingPayment :one
INSERT INTO incoming_payments (
    account_id,
    sender_name,
    sender_reference,
    amount,
    currency,
    status,
    external_reference
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;


-- name: GetIncomingPayment :one
SELECT *
FROM incoming_payments
WHERE id = $1
LIMIT 1;


-- name: ListIncomingPaymentsByAccountID :many
SELECT *
FROM incoming_payments
WHERE account_id = $1
ORDER BY created_at DESC;


-- name: UpdateIncomingPaymentStatus :one
UPDATE incoming_payments
SET
    status = $2::varchar,
    completed_at = CASE
        WHEN $2::varchar = 'completed'
        THEN now()
        ELSE completed_at
    END
WHERE id = $1
  AND status = 'pending'
RETURNING *;