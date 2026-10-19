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



-- name: LockUserForCardCreation :one
SELECT id
FROM users
WHERE id = $1
FOR UPDATE;

-- name: CountCardsByUserID :one
SELECT COUNT(*)
FROM cards
WHERE user_id = $1;