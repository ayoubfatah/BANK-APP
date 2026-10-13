package store

import (
	"context"
	"encoding/json"
	"net"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type CloseAccountTxParams struct {
	AccountID int64
	UserID    int64
	ActorID   int64
	ActorRole string
	IPAddress string
	UserAgent string
}

func (store *Store) CloseAccountTx(
	ctx context.Context,
	args CloseAccountTxParams,
) (int64, error) {
	var rowsAffected int64

	err := store.execTx(ctx, func(q *db.Queries) error {
		var err error

		rowsAffected, err = q.CloseAccount(ctx, db.CloseAccountParams{
			ID:     args.AccountID,
			UserID: args.UserID,
		})
		if err != nil {
			return err
		}

		if rowsAffected == 0 || args.ActorRole != "admin" {
			return nil
		}

		metadata, err := json.Marshal(map[string]string{
			"reason": "Admin-initiated account closure",
		})
		if err != nil {
			return err
		}

		ip := net.ParseIP(args.IPAddress)

		return q.CreateAuditLog(ctx, db.CreateAuditLogParams{
			UserID: pgtype.Int8{
				Int64: args.ActorID,
				Valid: true,
			},
			Action: "close_account",
			EntityType: pgtype.Text{
				String: "account",
				Valid:  true,
			},
			EntityID: pgtype.Int8{
				Int64: args.AccountID,
				Valid: true,
			},
			IpAddress: pgtype.Text{
				String: ip.String(),
				Valid:  ip != nil,
			},
			UserAgent: pgtype.Text{
				String: args.UserAgent,
				Valid:  true,
			},
			Metadata: metadata,
		})
	})

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
