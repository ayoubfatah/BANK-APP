-- name: CreateInvoice :one
INSERT INTO invoices (business_id, customer_user_id ,customer_name,customer_email,amount,currency,status,due_date,paid_at)
VALUES($1,$2,$3,$4,$5,$6,$7,$8, $9)
RETURNING * ;

-- name: GetInvoiceById :one
SELECT * FROM invoices 
WHERE id = $1 ;



-- name: ListInvoicesByBusinessID :many
SELECT * FROM invoices 
WHERE business_id = $1 
LIMIT $2
OFFSET $3;




-- name: UpdateInvoiceDetails :one
UPDATE invoices
SET
    customer_name = $2,
    customer_email = $3,
    due_date = $4
WHERE id = $1
  AND status = 'draft'
RETURNING *;


-- name: IssueInvoice :one
UPDATE invoices
SET
    status = 'pending'
WHERE id = $1
  AND status = 'draft'
RETURNING *;


-- name: MarkInvoiceAsPaid :one
UPDATE invoices
SET
    status = 'paid',
    paid_at = now()
WHERE id = $1
  AND status IN ('pending', 'overdue')
RETURNING *;


-- name: MarkInvoiceAsOverdue :many
UPDATE invoices
SET
    status = 'overdue'
WHERE status = 'pending'
  AND due_date < CURRENT_DATE
RETURNING *;


-- name: CancelInvoice :one
UPDATE invoices
SET
    status = 'cancelled'
WHERE id = $1
  AND status IN ('draft', 'pending', 'overdue')
RETURNING *;


-- name: DeleteInvoice :one
DELETE FROM invoices
WHERE id = $1
  AND status = 'draft'
RETURNING *;