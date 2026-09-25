-- name: CreateExchangeRate :one
INSERT INTO exchange_rates (
    base_currency,
    quote_currency,
    rate,
    source,
    fetched_at
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;


-- name: GetExchangeRate :one
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
  AND fetched_at = $3
LIMIT 1;


-- name: GetLatestExchangeRate :one
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
ORDER BY fetched_at DESC
LIMIT 1;


-- name: ListExchangeRates :many
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
ORDER BY fetched_at DESC
LIMIT $3
OFFSET $4;