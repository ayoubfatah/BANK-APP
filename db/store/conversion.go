package store

import (
	"context"
	"errors"
	"fmt"
	"math"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var (
	ErrSameAccountConversion  = errors.New("cannot convert to the same account")
	ErrCrossUserConversion    = errors.New("accounts must belong to the same user")
	ErrSameCurrencyConversion = errors.New("cannot convert a currency into itself")
)

type CreateConversionTransactionArgs struct {
	UserID         int64       `json:"user_id"`
	FromAccountID  int64       `json:"from_account_id"`
	ToAccountID    int64       `json:"to_account_id"`
	FromAmount     int64       `json:"from_amount"`
	Fee            int64       `json:"fee"`
	Status         string      `json:"status"`
	IdempotencyKey pgtype.Text `json:"idempotency_key"`
}

type CreateConversionTransactionResult struct {
	Conversion db.Conversion
	FromEntry  db.Entry
	ToEntry    db.Entry
}

func (store *SQLStore) CreateConversionTransaction(ctx context.Context, args CreateConversionTransactionArgs) (CreateConversionTransactionResult, error) {
	var result CreateConversionTransactionResult

	err := store.execTx(ctx, func(q *db.Queries) error {

		// 0. Idempotency check — same reasoning as transfers: a retried
		// request or double-tap shouldn't create a second conversion.
		if args.IdempotencyKey.Valid {
			existing, err := q.GetConversionByIdempotencyKey(ctx, args.IdempotencyKey)
			if err == nil {
				result.Conversion = existing
				return nil // already done, return the existing record, don't redo the work
			}
			if !errors.Is(err, pgx.ErrNoRows) {
				return err // a real DB error, not just "not found"
			}
		}

		if args.FromAccountID == args.ToAccountID {
			return ErrSameAccountConversion
		}

		// 1. Lock both accounts, lowest ID first — same rule as transfers.
		// Not strictly required for a single-user case, but cheap
		// insurance against the same user converting from two devices
		// at once.
		firstID, secondID := args.FromAccountID, args.ToAccountID
		if firstID > secondID {
			firstID, secondID = secondID, firstID
		}

		first, err := q.GetAccount(ctx, firstID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: account %d", ErrAccountNotFound, firstID)
			}
			return err
		}

		second, err := q.GetAccount(ctx, secondID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("%w: account %d", ErrAccountNotFound, secondID)
			}
			return err
		}

		// 2. Both accounts must belong to the same user — otherwise this
		// isn't a conversion, it's a disguised transfer between two
		// different people, which should go through TransferMoneyTx.
		if first.UserID != second.UserID {
			return ErrCrossUserConversion
		}

		// 3. Work out which is actually "from" and which is "to" —
		// sorting by ID may have scrambled the original direction.
		fromAccount, toAccount := first, second
		if first.ID != args.FromAccountID {
			fromAccount, toAccount = second, first
		}

		if fromAccount.Status != "active" {
			return fmt.Errorf("%w: source account %d is %s", ErrAccountNotActive, fromAccount.ID, fromAccount.Status)
		}
		if toAccount.Status != "active" {
			return fmt.Errorf("%w: destination account %d is %s", ErrAccountNotActive, toAccount.ID, toAccount.Status)
		}

		if fromAccount.Currency == toAccount.Currency {
			return ErrSameCurrencyConversion
		}

		totalDebit := args.FromAmount + args.Fee
		if fromAccount.Balance < totalDebit {
			return ErrInsufficientFunds
		}

		// 4. Look up the latest rate for this exact pair.
		exchangeRate, err := q.GetExchangeRate(ctx, db.GetExchangeRateParams{
			BaseCurrency:  fromAccount.Currency,
			QuoteCurrency: toAccount.Currency,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("no exchange rate found for %s -> %s", fromAccount.Currency, toAccount.Currency)
			}
			return err
		}

		rateFloat, err := exchangeRate.Rate.Float64Value()
		if err != nil {
			return fmt.Errorf("invalid exchange rate value: %w", err)
		}
		if !rateFloat.Valid {
			return fmt.Errorf("exchange rate for %s -> %s is null", fromAccount.Currency, toAccount.Currency)
		}

		// 5. Convert the amount, rounding to the nearest smallest unit
		// (cents). The fee is charged in the FROM currency and does not
		// get converted — it's simply not included in what gets converted.
		toAmount := int64(math.Round(float64(args.FromAmount) * rateFloat.Float64))

		// 6. Write the conversion record — this is the permanent receipt
		// of exactly what rate was used, frozen at this moment, separate
		// from the live exchange_rates table.
		result.Conversion, err = q.CreateConversion(ctx, db.CreateConversionParams{
			UserID:         args.UserID,
			FromAccountID:  fromAccount.ID,
			ToAccountID:    toAccount.ID,
			FromAmount:     args.FromAmount,
			ToAmount:       toAmount,
			FromCurrency:   fromAccount.Currency,
			ToCurrency:     toAccount.Currency,
			ExchangeRate:   exchangeRate.Rate,
			Fee:            args.Fee,
			Status:         "completed",
			IdempotencyKey: args.IdempotencyKey,
		})
		if err != nil {
			return err
		}

		// 7. Ledger entries. The fee stays out of the "to" side — it
		// never existed on that side of the conversion, it was consumed
		// as part of the debit on the "from" side.
		fromDescription := fmt.Sprintf("Converted %s to %s", fromAccount.Currency, toAccount.Currency)
		result.FromEntry, err = q.CreateEntry(ctx, db.CreateEntryParams{
			AccountID:   fromAccount.ID,
			Amount:      totalDebit,
			Currency:    fromAccount.Currency,
			Type:        "conversion",
			ReferenceID: pgtype.Int8{Int64: result.Conversion.ID, Valid: true},
			Description: pgtype.Text{String: fromDescription, Valid: true},
		})
		if err != nil {
			return err
		}

		toDescription := fmt.Sprintf("Converted from %s to %s", fromAccount.Currency, toAccount.Currency)
		result.ToEntry, err = q.CreateEntry(ctx, db.CreateEntryParams{
			AccountID:   toAccount.ID,
			Amount:      toAmount,
			Currency:    toAccount.Currency,
			Type:        "conversion",
			ReferenceID: pgtype.Int8{Int64: result.Conversion.ID, Valid: true},
			Description: pgtype.Text{String: toDescription, Valid: true},
		})
		if err != nil {
			return err
		}

		// 8. Move the actual balances.
		_, err = q.SubtractFromAccountBalance(ctx, db.SubtractFromAccountBalanceParams{
			Balance: totalDebit,
			ID:      fromAccount.ID,
		})
		if err != nil {
			return err
		}

		_, err = q.AddToAccountBalance(ctx, db.AddToAccountBalanceParams{
			Balance: toAmount,
			ID:      toAccount.ID,
		})
		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
