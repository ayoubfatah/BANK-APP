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


-- name: FreezeAccount :execrows 
UPDATE accounts 
SET status = 'frozen' ,
    freezed_by = $3 ,
    freezed_at = now(),
    updated_at = now()
WHERE id =$1 AND user_id = $2 and status = 'active'
RETURNING * ;

-- name: UnFreezeAccount :execrows
UPDATE accounts
SET
    status = 'active',
    freezed_by = NULL,
    freezed_at = NULL
WHERE
    id = $1
    AND user_id = $2
    AND status = 'frozen';

-- name: CloseAccount :execrows
UPDATE accounts
SET
    status = 'closed'
WHERE
    id = $1
    AND user_id = $2
    AND status = 'active';




-- name: ListAccountsAdmin :many
SELECT *
FROM accounts
WHERE (sqlc.narg('user_id')::bigint   IS NULL OR user_id  = sqlc.narg('user_id'))
  AND (sqlc.narg('currency')::text    IS NULL OR currency = sqlc.narg('currency'))
  AND (sqlc.narg('status')::text      IS NULL OR status   = sqlc.narg('status'))
ORDER BY id DESC
LIMIT sqlc.arg('limit')
OFFSET sqlc.arg('offset');
 