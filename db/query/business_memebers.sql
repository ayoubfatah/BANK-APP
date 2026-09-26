-- name: AddBusinessMember :one
INSERT INTO business_members (
    business_id,
    user_id,
    role
) VALUES (
    $1, $2, $3
)
RETURNING *;


-- name: GetBusinessMember :one
SELECT *
FROM business_members
WHERE id = $1
LIMIT 1;


-- name: ListBusinessMembers :many
SELECT *
FROM business_members
WHERE business_id = $1
ORDER BY created_at ASC;


-- name: UpdateBusinessMemberRole :one
UPDATE business_members
SET
    role = $2
WHERE id = $1
RETURNING *;


-- name: RemoveBusinessMember :exec
DELETE FROM business_members
WHERE id = $1;