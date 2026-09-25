-- name: CreateEntry :one
INSERT INTO entries (
    account_id,
    amount,
    currency,
    type,
    reference_id,
    description
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;


-- name: GetEntry :one
SELECT *
FROM entries
WHERE id = $1
LIMIT 1;


-- name: ListEntriesByAccountID :many
SELECT *
FROM entries
WHERE account_id = $1
ORDER BY created_at DESC, id DESC;


-- name: ListEntriesByAccountIDWithPagination :many
SELECT *
FROM entries
WHERE account_id = $1
ORDER BY created_at DESC, id DESC
LIMIT $2
OFFSET $3;


-- name: GetEntriesByReferenceID :many
SELECT *
FROM entries
WHERE reference_id = $1
ORDER BY created_at DESC, id DESC;


-- name: GetAccountBalanceFromEntries :one
SELECT COALESCE(
    SUM(
        CASE
            WHEN type = 'deposit' THEN amount
            WHEN type IN ('withdrawal', 'fee') THEN -amount
            ELSE 0
        END
    ),
    0
)::bigint AS balance
FROM entries
WHERE account_id = $1;