package store

import (
	"context"
	"errors"
	"fmt"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrIncomingPaymentAlreadyProcessed = errors.New("incoming payment already processed")
	ErrAccountNotActive                = errors.New("account is not active")
)

type CompleteIncomingPaymentTxArgs struct {
	ID          int64       `json:"id"`
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

		// 1. Lock the account first. Only one account is involved here,
		// so there's no ID-ordering rule needed (that's only for two
		// accounts belonging to different people, like a transfer).
		// But we still need the row locked before checking its status
		// or changing its balance, same reasoning as GetAccount in
		// CreateTransferTransaction.
		account, err := q.GetAccount(ctx, args.AccountID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: account %d", ErrAccountNotFound, args.AccountID)
			}
			return err
		}

		if account.Status != "active" {
			return fmt.Errorf("%w: account %d is %s", ErrAccountNotActive, account.ID, account.Status)
		}

		// 2. Mark incoming payment as completed.
		// WHERE status = 'pending' in the SQL means a retried webhook
		// finds zero matching rows and returns pgx.ErrNoRows here,
		// instead of double-crediting.
		result.IncomingPayment, err = q.UpdateIncomingPaymentStatus(
			ctx,
			db.UpdateIncomingPaymentStatusParams{
				ID:     args.ID,
				Column2: "completed",
			},
		)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: incoming payment %d is not pending", ErrIncomingPaymentAlreadyProcessed, args.ID)
			}
			return err
		}

		// 3. Create account entry
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

		// 4. Add money to account balance
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