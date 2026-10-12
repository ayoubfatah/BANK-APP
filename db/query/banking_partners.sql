-- name: CreateBankingPartner :one
INSERT INTO banking_partners (
    currency_code,
    country_code,
    provider_name,
    detail_type
)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetBankingPartnersByCurrencyAndCountry :many
SELECT *
FROM banking_partners
WHERE currency_code = $1
  AND country_code = $2
  AND is_active = TRUE
ORDER BY provider_name;

-- name: ListBankingPartners :many
SELECT *
FROM banking_partners
ORDER BY currency_code, country_code, provider_name;


-- name: GetBankingPartner :one 
SELECT * 
FROM banking_partners
WHERE id = $1 ;