-- name: CreateBeneficiary :one
INSERT INTO beneficiaries (
    user_id,
    name,
    account_number,
    iban,
    bic,
    bank_name,
    country,
    currency
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;


-- name: GetBeneficiary :one
SELECT *
FROM beneficiaries
WHERE id = $1
  AND user_id = $2
LIMIT 1;


-- name: ListBeneficiariesByUserID :many
SELECT *
FROM beneficiaries
WHERE user_id = $1
ORDER BY created_at DESC;


-- name: UpdateBeneficiary :one
UPDATE beneficiaries
SET
    name = $2,
    account_number = $3,
    iban = $4,
    bic = $5,
    bank_name = $6,
    country = $7,
    currency = $8,
    updated_at = NOW()
WHERE id = $1
  AND user_id = $9
RETURNING *;


-- name: DeleteBeneficiary :exec
DELETE FROM beneficiaries
WHERE id = $1
  AND user_id = $2;