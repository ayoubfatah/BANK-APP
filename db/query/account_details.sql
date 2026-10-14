-- name: CreateAccountDetail :one
INSERT INTO account_details(account_id , type, value , country , bank_name) 
VALUES ($1,$2,$3,$4 ,$5)
RETURNING * ;

-- name: GetAccountDetailByAccountId :one 
SELECT * FROM  account_details
WHERE account_id = $1
LIMIT 1;

-- name: GetAccountDetailById :one 
SELECT * FROM  account_details
WHERE id = $1
LIMIT 1 ;


-- name: ListAccountDetails :many
SELECT * FROM  account_details
LIMIT $1
OFFSET $2;

-- name: UpdateAccountDetail :exec
UPDATE account_details
SET
    type = $2,
    value = $3,
    country = $4,
    bank_name = $5
WHERE id = $1;


-- name: UpdateAccountDetailByAccountID :one
UPDATE account_details
SET
    type = $2,
    value = $3,
    country = $4,
    bank_name = $5
WHERE account_id = $1
RETURNING * ;



-- name: DeleteAccountDetail :exec
DELETE  FROM account_details 
WHERE id = $1 ;


