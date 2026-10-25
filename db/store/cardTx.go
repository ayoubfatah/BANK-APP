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

func createCardTx(
	ctx context.Context,
	q *db.Queries,
	arg CreateCardTxParams,
) (db.Card, db.CardSpendingLimit, error) {
	card, err := q.CreateCard(ctx, db.CreateCardParams{
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
	if err != nil {
		return db.Card{}, db.CardSpendingLimit{}, err
	}

	cardLimits, ok := defaults.DefaultCardLimits[arg.Currency]
	if !ok {
		cardLimits = defaults.FallbackCardLimits
	}

	limits, err := q.CreateCardSpendingLimits(
		ctx,
		db.CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: int64(cardLimits.Daily),
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: int64(cardLimits.Monthly),
				Valid: true,
			},
		},
	)
	if err != nil {
		return db.Card{}, db.CardSpendingLimit{}, err
	}

	return card, limits, nil
}

func (store *SQLStore) CreateCardTx(
	ctx context.Context,
	arg CreateCardTxParams,
) (db.Card, db.CardSpendingLimit, error) {
	var card db.Card
	var limits db.CardSpendingLimit

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

		card, limits, err = createCardTx(ctx, q, arg)
		return err
	})

	if err != nil {
		return db.Card{}, db.CardSpendingLimit{}, err
	}

	return card, limits, nil
}

// CreateCardTransaction inserts a new card_transactions row with
// status "authorized". Capture, void, or expiration transitions it later.
func (store *SQLStore) CreateCardTransactionTx(
	ctx context.Context,
	arg CreateCardTransactionParams,
) (db.CardTransaction, error) {
	return store.Queries.CreateCardTransaction(
		ctx,
		db.CreateCardTransactionParams{
			CardID:            arg.CardID,
			AccountID:         arg.AccountID,
			MerchantName:      arg.MerchantName,
			MerchantCategory:  arg.MerchantCategory,
			Amount:            arg.Amount,
			Currency:          arg.Currency,
			Status:            "authorized",
			AuthorizationCode: arg.AuthorizationCode,
		},
	)
}

func (store *SQLStore) CaptureCardTransactionTx(
	ctx context.Context,
	cardTxnID int64,
	finalAmount int64,
) (db.CardTransaction, error) {
	var result db.CardTransaction

	err := store.execTx(ctx, func(q *db.Queries) error {
		// 1. Change authorized -> captured and set the final amount.
		cardTxn, err := q.CaptureCardTransaction(
			ctx,
			db.CaptureCardTransactionParams{
				ID:     cardTxnID,
				Amount: finalAmount,
			},
		)
		if err != nil {
			return err
		}
		result = cardTxn

		// 2. Record the card transaction in the ledger.
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

		// 3. Subtract the captured amount from the account balance.
		_, err = q.SubtractFromAccountBalance(
			ctx,
			db.SubtractFromAccountBalanceParams{
				ID:      cardTxn.AccountID,
				Balance: finalAmount,
			},
		)
		return err
	})

	return result, err
}

// TODO: Add notification later.
func (store *SQLStore) VoidCardTransaction(
	ctx context.Context,
	id int64,
) (db.CardTransaction, error) {
	return store.Queries.VoidCardTransaction(ctx, id)
}

// TODO: Add notification later.
func (store *SQLStore) ExpireOldAuthorizations(
	ctx context.Context,
	maxAge pgtype.Interval,
) ([]db.CardTransaction, error) {
	return store.Queries.ExpireOldAuthorizations(ctx, maxAge)
}

type ReplaceCardTxArgs struct {
	OldCardID int64               `json:"old_card_id"`
	UserID    int64               `json:"user_id"`
	Reason    string              `json:"reason"`
	Currency  string              `json:"currency"`
	NewCard   db.CreateCardParams `json:"new_card"`
}

type ReplaceCardTxResult struct {
	OldCard db.Card
	NewCard db.Card
	Limits  db.CardSpendingLimit
}

// ReplaceCardTx cancels the old card and creates a replacement
// with its spending limits in the same transaction.
func (store *SQLStore) ReplaceCardTx(
	ctx context.Context,
	args ReplaceCardTxArgs,
) (ReplaceCardTxResult, error) {
	var result ReplaceCardTxResult

	err := store.execTx(ctx, func(q *db.Queries) error {
		// Cancel the old card first.
		oldCard, err := q.CancelCard(ctx, db.CancelCardParams{
			ID:     args.OldCardID,
			UserID: args.UserID,
			Reason: pgtype.Text{
				String: args.Reason,
				Valid:  true,
			},
		})
		if err != nil {
			return err
		}
		result.OldCard = oldCard

		// Create the replacement card and its spending limits.
		newCard, limits, err := createCardTx(
			ctx,
			q,
			CreateCardTxParams{
				UserID:     args.NewCard.UserID,
				AccountID:  args.NewCard.AccountID,
				CardNumber: args.NewCard.CardNumber,
				LastFour:   args.NewCard.LastFour,
				Cvv:        args.NewCard.Cvv,
				Brand:      args.NewCard.Brand,
				CardType:   args.NewCard.CardType,
				Status:     args.NewCard.Status,
				ExpiryDate: args.NewCard.ExpiryDate,
				Currency:   args.Currency,
			},
		)
		if err != nil {
			return err
		}

		// Link the new card to the replaced card.
		result.NewCard, err = q.SetCardReplacedCardID(
			ctx,
			db.SetCardReplacedCardIDParams{
				ID: newCard.ID,
				ReplacedCardID: pgtype.Int8{
					Int64: args.OldCardID,
					Valid: true,
				},
			},
		)
		if err != nil {
			return err
		}

		result.Limits = limits

		return nil
	})

	if err != nil {
		return ReplaceCardTxResult{}, err
	}

	return result, nil
}

//  Admin Block Card

func (store *SQLStore) blockCard() {}
