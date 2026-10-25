package store

import (
	"context"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type UpdateAccountDetailByAccountIDTxParams struct {
	Actor
	AccountID int64
	Type      string
	Value     string
	Country   pgtype.Text
	BankName  pgtype.Text
}

func (store *SQLStore) UpdateAccountDetailByAccountIDTx(
	ctx context.Context,
	args UpdateAccountDetailByAccountIDTxParams,
) (db.AccountDetail, error) {
	var updated db.AccountDetail

	err := store.execTx(ctx, func(q *db.Queries) error {
		var err error

		updated, err = q.UpdateAccountDetailByAccountID(ctx, db.UpdateAccountDetailByAccountIDParams{
			AccountID: args.AccountID,
			Type:      args.Type,
			Value:     args.Value,
			Country:   args.Country,
			BankName:  args.BankName,
		})
		if err != nil {
			return err
		}

		return writeAudit(ctx, q, args.Actor, AuditEntry{
			Action:     "update_account_detail",
			EntityType: "account_detail",
			EntityID:   updated.ID,
			Metadata: map[string]any{
				"reason":     "Admin updated account detail",
				"account_id": args.AccountID,
			},
		})
	})
	if err != nil {
		return db.AccountDetail{}, err
	}

	return updated, nil
}
