-- name: CreateCard :one
INSERT INTO cards (
    user_id,
    account_id,
    card_number,
    last_four,
    cvv,
    brand,
    card_type,
    status,
    expiry_date
) VALUES (
    sqlc.arg(user_id),
    sqlc.arg(account_id),
    sqlc.arg(card_number),
    sqlc.arg(last_four),
    sqlc.arg(cvv),
    sqlc.arg(brand),
    sqlc.arg(card_type),
    sqlc.arg(status),
    sqlc.arg(expiry_date)
)
RETURNING *;


-- name: GetCard :one
SELECT *
FROM cards
WHERE id = sqlc.arg(id)
LIMIT 1;


-- name: ListCardsByUserID :many
SELECT *
FROM cards
WHERE user_id = sqlc.arg(user_id)
ORDER BY created_at DESC;


-- name: ListCardsByAccountID :many
SELECT *
FROM cards
WHERE account_id = sqlc.arg(account_id)
ORDER BY created_at DESC;


-- name: UpdateCardStatus :one
UPDATE cards
SET
    status = sqlc.arg(status),
    updated_at = now()
WHERE id = sqlc.arg(id)
RETURNING *;


-- name: FreezeCard :one
UPDATE cards
SET status = 'frozen'
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'active'
RETURNING *;


-- name: UnFreezeCard :one
UPDATE cards
SET status = 'active'
WHERE id = sqlc.arg(id)
  AND user_id = sqlc.arg(user_id)
  AND status = 'frozen'
RETURNING *;


-- name: LockUserForCardCreation :one
SELECT id
FROM users
WHERE id = $1
FOR UPDATE;

-- name: CountCardsByUserID :one
SELECT COUNT(*)
FROM cards
WHERE user_id = $1
  AND status IN ('active', 'frozen', 'blocked');


-- name: GetCardByIDAndUserID :one
SELECT *
FROM cards
WHERE id = $1
  AND user_id = $2
LIMIT 1;


-- name: CancelCard :one
UPDATE cards
SET status = 'cancelled',
    cancelled_reason = sqlc.narg('reason'),
    cancelled_at = now(),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND status IN ('active', 'frozen')
RETURNING *;

-- name: AdminCancelCard :one
UPDATE cards
SET status = 'cancelled',
    cancelled_reason = sqlc.narg('reason'),
    cancelled_at = now(),
    updated_at = now()
WHERE id = sqlc.arg('id')
  AND status IN ('active', 'frozen', 'blocked')
RETURNING *;