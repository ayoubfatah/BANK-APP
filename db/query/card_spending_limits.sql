-- name: CreateCardSpendingLimits :one
INSERT INTO card_spending_limits (
    card_id,
    daily_limit,
    monthly_limit
)
VALUES (
    sqlc.arg(card_id),
    sqlc.narg(daily_limit),
    sqlc.narg(monthly_limit)
)
RETURNING *;


-- name: GetCardSpendingLimits :one
SELECT csl.*
FROM card_spending_limits csl
JOIN cards c ON c.id = csl.card_id
WHERE csl.card_id = sqlc.arg(card_id)
  AND c.user_id = sqlc.arg(user_id);

-- name: UpdateCardSpendingLimits :one
UPDATE card_spending_limits AS csl
SET
    daily_limit = sqlc.narg(daily_limit),
    monthly_limit = sqlc.narg(monthly_limit),
    updated_at = NOW()
FROM cards AS c
WHERE csl.card_id = sqlc.arg(card_id)
  AND c.id = csl.card_id
  AND c.user_id = sqlc.arg(user_id)
RETURNING csl.*;