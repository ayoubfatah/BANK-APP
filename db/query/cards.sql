-- name: CreateCard :one
INSERT INTO cards (
    user_id,
    account_id,
    last_four,
    card_type,
    status,
    expiry_date
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;


-- name: GetCard :one
SELECT *
FROM cards
WHERE id = $1
LIMIT 1;


-- name: ListCardsByUserID :many
SELECT *
FROM cards
WHERE user_id = $1
ORDER BY created_at DESC;


-- name: ListCardsByAccountID :many
SELECT *
FROM cards
WHERE account_id = $1
ORDER BY created_at DESC;


-- name: UpdateCardStatus :one
UPDATE cards
SET
    status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;