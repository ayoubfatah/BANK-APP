package store

import (
	"context"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateCardTransactionParams struct {
	CardID            int64
	AccountID         int64
	MerchantName      string
	MerchantCategory  pgtype.Text
	Amount            int64
	Currency          string
	AuthorizationCode pgtype.Text
}

// CreateCardTransaction inserts a new card_transactions row with
// status "authorized" — this is the hold. Capture/void/expire later
// transition it based on that status.
func (store *Store) CreateCardTransaction(ctx context.Context, arg CreateCardTransactionParams) (db.CardTransaction, error) {
	return store.Queries.CreateCardTransaction(ctx, db.CreateCardTransactionParams{
		CardID:            arg.CardID,
		AccountID:         arg.AccountID,
		MerchantName:      arg.MerchantName,
		MerchantCategory:  arg.MerchantCategory,
		Amount:            arg.Amount,
		Currency:          arg.Currency,
		Status:            "authorized",
		AuthorizationCode: arg.AuthorizationCode,
	})
}

func (store *Store) CaptureCardTransactionTx(ctx context.Context, cardTxnID int64, finalAmount int64) (db.CardTransaction, error) {
	var result db.CardTransaction

	err := store.execTx(ctx, func(q *db.Queries) error {
		// 1. Flip status authorized -> captured, correct the amount to the real final charge
		cardTxn, err := q.CaptureCardTransaction(ctx, db.CaptureCardTransactionParams{
			ID:     cardTxnID,
			Amount: finalAmount,
		})
		if err != nil {
			return err
		}
		result = cardTxn

		// 2. Log it — this is just a record, does NOT touch balance
		_, err = q.CreateEntry(ctx, db.CreateEntryParams{
			AccountID:   cardTxn.AccountID,
			Amount:      finalAmount,
			Currency:    cardTxn.Currency,
			Type:        "card",
			ReferenceID: pgtype.Int8{Int64: cardTxn.ID, Valid: true},
		})
		if err != nil {
			return err
		}

		// 3. THIS is what actually moves the number — separate, explicit call
		_, err = q.SubtractFromAccountBalance(ctx, db.SubtractFromAccountBalanceParams{
			ID:      cardTxn.AccountID,
			Balance: finalAmount,
		})
		return err
	})

	return result, err
}

// TODO :  later on will add notification
func (store *Store) VoidCardTransaction(ctx context.Context, id int64) (db.CardTransaction, error) {
	return store.Queries.VoidCardTransaction(ctx, id)
}

// TODO :  later on will add notification
func (store *Store) ExpireOldAuthorizations(ctx context.Context, maxAge pgtype.Interval) ([]db.CardTransaction, error) {
	return store.Queries.ExpireOldAuthorizations(ctx, maxAge)
}
