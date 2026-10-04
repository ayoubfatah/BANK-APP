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
SELECT * 
FROM kyc_verifications 
LIMIT $1 
OFFSET $2;


-- name: VerifyKYC :one 
UPDATE kyc_verifications
SET status = $1 ,
    verified_at = NOW()
WHERE id = $2
RETURNING * ;


-- name: RejectKYC :one 
UPDATE kyc_verifications
SET
    status = 'rejected',
    rejection_reason = $2
WHERE id = $1
RETURNING *;



-- name: GetLatestKYCByUser :one
SELECT id, user_id, document_type, document_number, status, rejection_reason, verified_at, created_at
FROM kyc_verifications
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;