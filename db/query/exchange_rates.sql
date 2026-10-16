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
ORDER BY id DESC
LIMIT 1;


-- name: ListExchangeRates :many
SELECT *
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
ORDER BY id DESC
LIMIT $3
OFFSET $4;


-- name: GetLatestExchangeRate :one
SELECT id, base_currency, quote_currency, rate
FROM exchange_rates
WHERE base_currency = $1
  AND quote_currency = $2
ORDER BY id DESC
LIMIT 1;


-- name: ListLatestExchangeRates :many
SELECT DISTINCT ON (er.base_currency, er.quote_currency)
    er.id,
    er.base_currency,
    er.quote_currency,
    er.rate
FROM exchange_rates er
JOIN currencies f
    ON f.code = er.base_currency
   AND f.is_active
JOIN currencies t
    ON t.code = er.quote_currency
   AND t.is_active
ORDER BY
    er.base_currency,
    er.quote_currency,
    er.id DESC;


-- name: ListExchangeRateHistory :many
SELECT id, base_currency, quote_currency, rate
FROM exchange_rates
WHERE base_currency = sqlc.arg('base_currency')
  AND quote_currency = sqlc.arg('quote_currency')
ORDER BY id ASC;


-- name: ListExchangeRatesAdmin :many
SELECT id, base_currency, quote_currency, rate
FROM exchange_rates
WHERE (
    sqlc.narg('base_currency')::text IS NULL
    OR base_currency = sqlc.narg('base_currency')
)
AND (
    sqlc.narg('quote_currency')::text IS NULL
    OR quote_currency = sqlc.narg('quote_currency')
)
ORDER BY id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');