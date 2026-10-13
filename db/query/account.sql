-- name: CreateAccount :one
INSERT INTO accounts(user_id , currency , balance , status )
VALUES($1,$2,$3,$4)
RETURNING * ;


-- name: GetAccount :one 
SELECT * FROM accounts 
WHERE id = $1
FOR NO KEY UPDATE 
 ;

-- name: GetAccountByUserAndCurrency :one 
SELECT * FROM accounts 
WHERE  user_id = $1 AND currency = $2;

-- name: GetAccountByIDAndUserID :one 
SELECT * FROM accounts 
where id = $1 AND  user_id = $2 ;


-- name: ListAccountsByUserID :many
SELECT * FROM accounts 
WHERE user_id = $1;


-- name: UpdateAccountStatus :one 
UPDATE accounts 
SET status = $1 ,
    updated_at = now()
WHERE id =$2
RETURNING * ;

-- name: UpdateAccountBalance :one
UPDATE accounts
SET balance = $1,
    updated_at = now()
WHERE id = $2
RETURNING *;

-- name: AddToAccountBalance :one

UPDATE accounts
SET balance = balance + $1,
    updated_at = now()
WHERE id = $2
RETURNING *;

-- name: SubtractFromAccountBalance :one

UPDATE accounts
SET balance = balance - $1,
    updated_at = now()
WHERE id = $2
RETURNING *;


-- name: FreezeAccount :exec 
UPDATE accounts 
SET status = 'frozen' ,
    updated_at = now()
WHERE id =$1 AND user_id = $2
RETURNING * ;