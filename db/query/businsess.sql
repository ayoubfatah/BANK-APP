
-- name: CreateBusiness :one
INSERT INTO businesses (
    owner_id,
    name,
    registration_number,
    country,
    status
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5
)
RETURNING *;


-- name: GetBusiness :one
SELECT *
FROM businesses
WHERE id = $1
LIMIT 1;


-- name: ListBusinessesByOwnerID :many
SELECT *
FROM businesses
WHERE owner_id = $1
ORDER BY created_at DESC;


-- name: UpdateBusiness :one
UPDATE businesses
SET
    name = $2,
    registration_number = $3,
    country = $4,
    updated_at = now()
WHERE id = $1
RETURNING *;


-- name: UpdateBusinessStatus :one
UPDATE businesses
SET
    status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;