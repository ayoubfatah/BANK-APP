-- name: CreateCurrency :one
INSERT INTO currencies (
    code,
    name,
    symbol,
    decimals,
    is_active
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;


-- name: GetCurrency :one
SELECT *
FROM currencies
WHERE code = $1
LIMIT 1;


-- name: ListCurrencies :many
SELECT *
FROM currencies
ORDER BY code
LIMIT $1
OFFSET $2;


-- name: UpdateCurrency :one
UPDATE currencies
SET
    name = $2,
    symbol = $3,
    decimals = $4
WHERE code = $1
RETURNING *;



-- name: DeleteAllCurrencies :exec
DELETE FROM currencies;



-- name: UpdateCurrencyActiveState :one
UPDATE currencies
SET
    is_active= $2
WHERE code = $1
RETURNING *;