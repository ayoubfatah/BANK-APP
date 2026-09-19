-- name: CreateUser :one
INSERT INTO users (
  first_name , last_name , email , phone , password_hash  , status , created_at , updated_at
) VALUES (
  $1, $2, $3 ,$4 ,$5 ,$6 ,&7,&8
)
RETURNING *;


-- name: GetUser :one
SELECT * FROM users 
WHERE id = $1 LIMIT 1;


-- name: GetAllUsers :many
SELECT * FROM users 
ORDER BY id
LIMIT $1
OFFSET $2;

-- name: UpdateUser :one
UPDATE users
SET
    first_name = $2,
    last_name = $3,
    updated_at = now()
WHERE id = $1
RETURNING *;


-- name: UpdateUserEmail :one
UPDATE users
SET
    email = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpdateUserPhone :exec
UPDATE users
SET
    phone = $2,
    updated_at = now()
WHERE id = $1;

-- name: UpdateUserStatus :one
UPDATE users
SET
    status = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;


-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;