-- name: CreatePaymentLink :one
INSERT INTO payment_links (
    business_id,
    token,
    amount,
    currency,
    description,
    status,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7
)
RETURNING *;


-- name: GetPaymentLink :one
SELECT *
FROM payment_links
WHERE id = $1
LIMIT 1;


-- name: GetPaymentLinkByToken :one
SELECT *
FROM payment_links
WHERE token = $1
LIMIT 1;


-- name: ListPaymentLinksByBusinessID :many
SELECT *
FROM payment_links
WHERE business_id = $1
ORDER BY created_at DESC;


-- name: UpdatePaymentLinkStatus :one
UPDATE payment_links
SET status = $2
WHERE id = $1
  AND (
      $2 != 'paid'
      OR expires_at IS NULL
      OR expires_at > now()
  )
RETURNING *;


-- name: DeletePaymentLink :exec
DELETE FROM payment_links
WHERE id = $1;