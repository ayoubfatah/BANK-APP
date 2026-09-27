package store

import (
	"context"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)
type CompleteIncomingPaymentTxArgs struct {
	ID          int64       `json:"id"`
	Balance     int64       `json:"balance"`
	AccountID   int64       `json:"account_id"`
	Amount      int64       `json:"amount"`
	Currency    string      `json:"currency"`
	Type        string      `json:"type"`
	ReferenceID pgtype.Int8 `json:"reference_id"`
	Description pgtype.Text `json:"description"`
}

type CompleteIncomingPaymentTxResult struct {
	IncomingPayment db.IncomingPayment
	Entry           db.Entry
	Account         db.Account
}

func (store *Store) CompleteIncomingPaymentTx(
	ctx context.Context,
	args CompleteIncomingPaymentTxArgs,
) (CompleteIncomingPaymentTxResult, error) {

	var result CompleteIncomingPaymentTxResult

	err := store.execTx(ctx, func(q *db.Queries) error {
		var err error

		// 1. Mark incoming payment as completed
		result.IncomingPayment, err = q.UpdateIncomingPaymentStatus(
			ctx,
			db.UpdateIncomingPaymentStatusParams{
				ID:     args.ID,
				Status: "completed",
			},
		)

		if err != nil {
			return err
		}

		// 2. Create account entry
		result.Entry, err = q.CreateEntry(
			ctx,
			db.CreateEntryParams{
				AccountID:   args.AccountID,
				Amount:      args.Amount,
				Currency:    args.Currency,
				Type:        args.Type,
				ReferenceID: args.ReferenceID,
				Description: args.Description,
			},
		)

		if err != nil {
			return err
		}

		// 3. Add money to account balance
		result.Account, err = q.AddToAccountBalance(
			ctx,
			db.AddToAccountBalanceParams{
				Balance: args.Amount,
				ID:      args.AccountID,
			},
		)

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}