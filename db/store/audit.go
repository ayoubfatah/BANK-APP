// store/audit.go
package store

import (
	"context"
	"encoding/json"
	"net"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type Actor struct {
	ID        int64
	Role      string
	IPAddress string
	UserAgent string
}

func (a Actor) IsAdmin() bool { return a.Role == "admin" }

type AuditEntry struct {
	Action     string
	EntityType string
	EntityID   int64
	Metadata   map[string]any
}

func writeAudit(ctx context.Context, q *db.Queries, actor Actor, e AuditEntry) error {
	if e.Metadata == nil {
		e.Metadata = map[string]any{}
	}

	raw, err := json.Marshal(e.Metadata)
	if err != nil {
		return err
	}

	var ipText pgtype.Text
	if ip := net.ParseIP(actor.IPAddress); ip != nil {
		ipText = pgtype.Text{String: ip.String(), Valid: true}
	}

	return q.CreateAuditLog(ctx, db.CreateAuditLogParams{
		UserID:     pgtype.Int8{Int64: actor.ID, Valid: true},
		Action:     e.Action,
		EntityType: pgtype.Text{String: e.EntityType, Valid: e.EntityType != ""},
		EntityID:   pgtype.Int8{Int64: e.EntityID, Valid: e.EntityID != 0},
		IpAddress:  ipText,
		UserAgent:  pgtype.Text{String: actor.UserAgent, Valid: actor.UserAgent != ""},
		Metadata:   raw,
	})
}
