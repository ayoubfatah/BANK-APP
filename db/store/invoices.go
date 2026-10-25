package store

import (
	"context"
	"errors"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

type MarkInvoiceAsPaidTransactionParams struct {
	InvoiceID     int64  `json:"invoice_id"`
	FromAccountID int64  `json:"from_account_id"`
	ToAccountID   int64  `json:"to_account_id"`
	Currency      string `json:"currency"`
	Amount        int64  `json:"amount"`
	Description   string `json:"description"`
}

func (store *SQLStore) MarkInvoiceAsPaidTransaction(
	ctx context.Context,
	arg MarkInvoiceAsPaidTransactionParams,
) error {

	var ErrInsufficientFunds = errors.New("insufficient funds")

	err := store.execTx(ctx, func(q *db.Queries) error {

		// --------------------------------------------------
		// 1. Lock both accounts in a consistent order
		// --------------------------------------------------

		firstID, secondID := arg.FromAccountID, arg.ToAccountID

		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}

		first, err := q.GetAccount(ctx, firstID)
		if err != nil {
			return err
		}

		second, err := q.GetAccount(ctx, secondID)
		if err != nil {
			return err
		}

		// --------------------------------------------------
		// 2. Work out which account is sender/receiver
		// --------------------------------------------------

		from, to := first, second

		if first.ID != arg.FromAccountID {
			from, to = second, first
		}

		// --------------------------------------------------
		// 3. Check balance using the locked account
		// --------------------------------------------------

		if from.Balance < arg.Amount {
			return ErrInsufficientFunds
		}

		// --------------------------------------------------
		// 4. Subtract money from sender
		// --------------------------------------------------

		_, err = q.SubtractFromAccountBalance(
			ctx,
			db.SubtractFromAccountBalanceParams{
				Balance: arg.Amount,
				ID:      from.ID,
			},
		)

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// 5. Add money to receiver
		// --------------------------------------------------

		_, err = q.AddToAccountBalance(
			ctx,
			db.AddToAccountBalanceParams{
				Balance: arg.Amount,
				ID:      to.ID,
			},
		)

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// 6. Create sender entry
		// --------------------------------------------------

		fromDescription := arg.Description + " to account"

		_, err = q.CreateEntry(
			ctx,
			db.CreateEntryParams{
				AccountID: from.ID,
				Amount:    arg.Amount,
				Currency:  arg.Currency,
				Type:      "withdrawal",
				Description: pgtype.Text{
					String: fromDescription,
					Valid:  true,
				},
			},
		)

		if err != nil {
			return err
		}

		// --------------------------------------------------
		// 7. Create receiver entry
		// --------------------------------------------------

		toDescription := arg.Description + " from account"

		_, err = q.CreateEntry(
			ctx,
			db.CreateEntryParams{
				AccountID: to.ID,
				Amount:    arg.Amount,
				Currency:  arg.Currency,
				Type:      "deposit",
				Description: pgtype.Text{
					String: toDescription,
					Valid:  true,
				},
			},
		)

		if err != nil {
			return err
		}

		_, err = q.MarkInvoiceAsPaid(context.Background(), arg.InvoiceID)
		if err != nil {
			return err
		}

		return nil
	})

	return err
}
