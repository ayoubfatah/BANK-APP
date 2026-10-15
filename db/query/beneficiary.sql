-- name: CreateBeneficiary :one
INSERT INTO beneficiaries (
    user_id,
    name,
    nickname,
    account_number,
    iban,
    bic,
    routing_number,
    bank_name,
    country,
    currency
) VALUES (
    sqlc.arg('user_id'),
    sqlc.arg('name'),
    sqlc.narg('nickname'),
    sqlc.narg('account_number'),
    sqlc.narg('iban'),
    sqlc.narg('bic'),
    sqlc.narg('routing_number'),
    sqlc.arg('bank_name'),
    sqlc.arg('country'),
    sqlc.arg('currency')
)
RETURNING *;


-- name: GetBeneficiary :one
SELECT *
FROM beneficiaries
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND deleted_at IS NULL
LIMIT 1;


-- name: ListBeneficiariesByUserID :many
SELECT *
FROM beneficiaries
WHERE user_id = sqlc.arg('user_id')
  AND deleted_at IS NULL
  AND (
    sqlc.narg('search')::text IS NULL
    OR name           ILIKE '%' || sqlc.narg('search')::text || '%'
    OR nickname       ILIKE '%' || sqlc.narg('search')::text || '%'
    OR bank_name      ILIKE '%' || sqlc.narg('search')::text || '%'
    OR iban           ILIKE '%' || sqlc.narg('search')::text || '%'
    OR account_number ILIKE '%' || sqlc.narg('search')::text || '%'
  )
ORDER BY created_at DESC;


-- name: UpdateBeneficiary :one
UPDATE beneficiaries
SET
    name           = COALESCE(sqlc.narg('name'), name),
    nickname       = COALESCE(sqlc.narg('nickname'), nickname),
    account_number = COALESCE(sqlc.narg('account_number'), account_number),
    iban           = COALESCE(sqlc.narg('iban'), iban),
    bic            = COALESCE(sqlc.narg('bic'), bic),
    routing_number = COALESCE(sqlc.narg('routing_number'), routing_number),
    bank_name      = COALESCE(sqlc.narg('bank_name'), bank_name),
    country        = COALESCE(sqlc.narg('country'), country),
    currency       = COALESCE(sqlc.narg('currency'), currency),
    updated_at     = NOW()
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND deleted_at IS NULL
RETURNING *;


-- name: DeleteBeneficiary :execrows
UPDATE beneficiaries
SET deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg('id')
  AND user_id = sqlc.arg('user_id')
  AND deleted_at IS NULL;