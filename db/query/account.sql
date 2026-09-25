-- name: CreateAccount :one
INSERT INTO accounts(user_id , currency , balance , status )
VALUES($1,$2,$3,$4)
RETURNING * ;


-- name: GetAccount :one 
SELECT * FROM accounts 
WHERE id = $1 ;

-- name: GetAccountByUserAndCurrency :one 
SELECT * FROM accounts 
WHERE  user_id = $1 AND currency = $2;



-- name: ListAccountsByUserID :many
SELECT * FROM accounts 
WHERE user_id = $1;


-- name: UpdateAccountStatus :one 
UPDATE accounts 
SET status = $1 ,
    updated_at = now()
WHERE id =$2
RETURNING * ;




