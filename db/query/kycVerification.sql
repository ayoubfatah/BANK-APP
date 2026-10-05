-- name: CreateKYCVerification :one
INSERT INTO kyc_verifications (
    user_id,
    document_type,
    document_number,
    status,
    rejection_reason,
    verified_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;





-- name: GetKYCVerificationByUserId :many
SELECT * 
FROM kyc_verifications
WHERE user_id = $1 ;


-- name: GetKYCVerification :one 
SELECT *
FROM kyc_verifications 
WHERE id = $1 LIMIT 1 ;


-- name: ListKYCVerifications :many
SELECT * FROM kyc_verifications
WHERE (sqlc.arg(status)::text = '' OR status = sqlc.arg(status)::text)
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg(page_size) OFFSET sqlc.arg(page_offset);


-- name: VerifyKYC :one
UPDATE kyc_verifications
SET status = 'verified', verified_at = NOW(), rejection_reason = NULL
WHERE id = $1 AND status = 'pending'
RETURNING *;


-- name: RejectKYC :one 
UPDATE kyc_verifications
SET status = 'rejected', rejection_reason = $2, verified_at = NULL
WHERE id = $1 AND status = 'pending'
RETURNING *;



-- name: GetLatestKYCByUser :one
SELECT id, user_id, document_type, document_number, status, rejection_reason, verified_at, created_at
FROM kyc_verifications
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;