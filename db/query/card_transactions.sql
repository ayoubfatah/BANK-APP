-- name: CreateCardTransaction :one
INSERT INTO card_transactions (
  card_id,
  account_id,
  merchant_name,
  merchant_category,
  amount,
  currency,
  status,
  authorization_code
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetCardTransaction :one
SELECT * FROM card_transactions
WHERE id = $1;

-- name: GetCardTransactionForUpdate :one
SELECT * FROM card_transactions
WHERE id = $1
FOR UPDATE;

-- name: CaptureCardTransaction :one
UPDATE card_transactions
SET status = 'captured',
    amount = $2
WHERE id = $1 AND status = 'authorized'
RETURNING *;

-- name: VoidCardTransaction :one
UPDATE card_transactions
SET status = 'voided'
WHERE id = $1 AND status = 'authorized'
RETURNING *;

-- name: ExpireOldAuthorizations :many
UPDATE card_transactions
SET status = 'expired'
WHERE status = 'authorized'
  AND created_at < now() - sqlc.arg(max_age)::interval
RETURNING *;

-- name: GetHeldAmount :one
SELECT COALESCE(SUM(amount), 0)::bigint AS held_amount
FROM card_transactions
WHERE account_id = $1 AND status = 'authorized';

-- name: GetAvailableBalance :one
SELECT
  a.balance,
  a.balance - COALESCE((
    SELECT SUM(ct.amount)
    FROM card_transactions ct
    WHERE ct.account_id = a.id AND ct.status = 'authorized'
  ), 0)::bigint AS available_balance
FROM accounts a
WHERE a.id = $1;

-- name: ListCardTransactionsByAccount :many
SELECT * FROM card_transactions
WHERE account_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;




-- name: GetCardDailySpending :one
SELECT COALESCE(SUM(ct.amount), 0)::BIGINT
FROM card_transactions ct
JOIN accounts a ON a.id = ct.account_id
WHERE ct.card_id = sqlc.arg(card_id)
 AND ct.status IN ('authorized', 'captured')
  AND ct.currency = a.currency
  AND ct.created_at >= (
      date_trunc(
          'day',
          NOW() AT TIME ZONE 'Africa/Casablanca'
      ) AT TIME ZONE 'Africa/Casablanca'
  )
  AND ct.created_at < (
      (
          date_trunc(
              'day',
              NOW() AT TIME ZONE 'Africa/Casablanca'
          ) + INTERVAL '1 day'
      ) AT TIME ZONE 'Africa/Casablanca'
  ); 

-- name: GetCardMonthlySpending :one
SELECT COALESCE(SUM(ct.amount), 0)::BIGINT
FROM card_transactions ct
JOIN accounts a ON a.id = ct.account_id
WHERE ct.card_id = sqlc.arg(card_id)
 AND ct.status IN ('authorized', 'captured')
  AND ct.currency = a.currency
  AND ct.created_at >= (
      date_trunc(
          'month',
          NOW() AT TIME ZONE 'Africa/Casablanca'
      ) AT TIME ZONE 'Africa/Casablanca'
  )
  AND ct.created_at < (
      (
          date_trunc(
              'month',
              NOW() AT TIME ZONE 'Africa/Casablanca'
          ) + INTERVAL '1 month'
      ) AT TIME ZONE 'Africa/Casablanca'
  );



  -- name: DeleteCardSpendingLimits :execrows
DELETE FROM card_spending_limits csl
USING cards c
WHERE csl.card_id = sqlc.arg(card_id)
  AND c.id = csl.card_id
  AND c.user_id = sqlc.arg(user_id);