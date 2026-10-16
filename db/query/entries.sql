
-- name: CreateEntry :one
INSERT INTO entries (
    account_id,
    amount,
    currency,
    type,
    reference_id,
    description
) VALUES (
    sqlc.arg('account_id'),
    sqlc.arg('amount'),
    sqlc.arg('currency'),
    sqlc.arg('type'),
    sqlc.narg('reference_id'),
    sqlc.narg('description')
)
RETURNING *;


-- name: GetEntry :one
SELECT *
FROM entries
WHERE id = $1
LIMIT 1;


-- name: GetEntryByAccountID :one
SELECT
	e.* , a.user_id
FROM
	entries e
	JOIN accounts a ON e.account_id = a.id
WHERE
	a.id = sqlc.arg('account_id') AND  e.id = sqlc.arg('id')
LIMIT 1 ;

-- name: ListEntriesByAccountID :many
SELECT *
FROM entries
WHERE account_id = $1
ORDER BY created_at DESC, id DESC;


-- name: ListEntriesByAccountIDWithPagination :many
SELECT *
FROM entries
WHERE account_id = sqlc.arg('account_id')
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');


-- name: ListEntriesByAccountIDFiltered :many
SELECT *
FROM entries
WHERE account_id = sqlc.arg('account_id')
  AND (
    sqlc.narg('from_date')::timestamptz IS NULL
    OR created_at >= sqlc.narg('from_date')::timestamptz
  )
  AND (
    sqlc.narg('to_date')::timestamptz IS NULL
    OR created_at <= sqlc.narg('to_date')::timestamptz
  )
  AND (
    sqlc.narg('entry_type')::varchar IS NULL
    OR type = sqlc.narg('entry_type')::varchar
  )
  AND (
    sqlc.narg('min_amount')::bigint IS NULL
    OR amount >= sqlc.narg('min_amount')::bigint
  )
  AND (
    sqlc.narg('max_amount')::bigint IS NULL
    OR amount <= sqlc.narg('max_amount')::bigint
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('page_size')
OFFSET sqlc.arg('offset');


-- name: GetEntriesByReferenceID :many
SELECT *
FROM entries
WHERE reference_id = sqlc.arg('reference_id')
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


-- name: GetAccountOwnerAndBalance :one
SELECT id, user_id, balance
FROM accounts
WHERE id = $1
LIMIT 1;
