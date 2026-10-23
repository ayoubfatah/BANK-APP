package store

import (
	"context"
	"errors"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/defaults"
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

type CreateCardTxParams struct {
	UserID     int64
	AccountID  int64
	CardNumber string
	LastFour   string
	Cvv        string
	Brand      string
	CardType   string
	Status     string
	ExpiryDate pgtype.Date
	Currency   string
}

var ErrCardLimitReached = errors.New("card limit per user reached")

func (store *Store) CreateCardTx(
	ctx context.Context,
	arg CreateCardTxParams,
) (db.Card, error) {
	var card db.Card

	err := store.execTx(ctx, func(q *db.Queries) error {
		// Lock the user row so concurrent card creations
		// for this user are serialized.
		_, err := q.LockUserForCardCreation(ctx, arg.UserID)
		if err != nil {
			return err
		}

		count, err := q.CountCardsByUserID(ctx, arg.UserID)
		if err != nil {
			return err
		}

		if count >= 4 {
			return ErrCardLimitReached
		}

		card, err = q.CreateCard(ctx, db.CreateCardParams{
			UserID:     arg.UserID,
			AccountID:  arg.AccountID,
			CardNumber: arg.CardNumber,
			LastFour:   arg.LastFour,
			Cvv:        arg.Cvv,
			Brand:      arg.Brand,
			CardType:   arg.CardType,
			Status:     arg.Status,
			ExpiryDate: arg.ExpiryDate,
		})

		limits, ok := defaults.DefaultCardLimits[arg.Currency]
		if !ok {
			limits = defaults.FallbackCardLimits
		}

		_, err = q.CreateCardSpendingLimits(ctx, db.CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: int64(limits.Daily),
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: int64(limits.Daily),
				Valid: true,
			},
		})

		if err != nil {
			return err
		}

		return err
	})

	if err != nil {
		return db.Card{}, err
	}

	return card, nil
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
