-- name: CreateExchangeRate :one
INSERT INTO exchange_rates (
    base_currency,
    quote_currency,
    rate,
    source
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;


-- name: GetExchangeRate :one
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
LIMIT 1;


-- name: ListExchangeRates :many
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
ORDER BY id DESC
LIMIT $3
OFFSET $4;