-- name: CreateAuditLog :exec
INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip_address, user_agent, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7);