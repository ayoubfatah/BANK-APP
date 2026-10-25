package store

import (
	"context"
	"errors"
	"fmt"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrConversionNotReversible = errors.New("conversion cannot be reversed")

type ReverseConversionTransactionArgs struct {
	ConversionID int64  `json:"conversion_id"`
	Reason       string `json:"reason"`
}

func (store *SQLStore) ReverseConversionTransaction(ctx context.Context, args ReverseConversionTransactionArgs) (db.Conversion, error) {
	var reversed db.Conversion

	err := store.execTx(ctx, func(q *db.Queries) error {
		conversion, err := q.GetConversionForUpdate(ctx, args.ConversionID)
		if err != nil {
			return err
		}
		if conversion.Status != "completed" {
			return ErrConversionNotReversible
		}

		firstID, secondID := conversion.FromAccountID, conversion.ToAccountID
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

		fromAccount, toAccount := first, second
		if first.ID != conversion.FromAccountID {
			fromAccount, toAccount = second, first
		}

		// the user gets back what they paid, fee included
		refund := conversion.FromAmount + conversion.Fee

		if toAccount.Balance < conversion.ToAmount {
			return ErrInsufficientFunds
		}

		_, err = q.SubtractFromAccountBalance(ctx, db.SubtractFromAccountBalanceParams{
			Balance: conversion.ToAmount,
			ID:      toAccount.ID,
		})
		if err != nil {
			return err
		}

		_, err = q.AddToAccountBalance(ctx, db.AddToAccountBalanceParams{
			Balance: refund,
			ID:      fromAccount.ID,
		})
		if err != nil {
			return err
		}

		description := fmt.Sprintf("Reversal of conversion #%d: %s", conversion.ID, args.Reason)

		_, err = q.CreateEntry(ctx, db.CreateEntryParams{
			AccountID:   toAccount.ID,
			Amount:      conversion.ToAmount,
			Currency:    toAccount.Currency,
			Type:        "conversion",
			ReferenceID: pgtype.Int8{Int64: conversion.ID, Valid: true},
			Description: pgtype.Text{String: description, Valid: true},
		})
		if err != nil {
			return err
		}

		_, err = q.CreateEntry(ctx, db.CreateEntryParams{
			AccountID:   fromAccount.ID,
			Amount:      refund,
			Currency:    fromAccount.Currency,
			Type:        "conversion",
			ReferenceID: pgtype.Int8{Int64: conversion.ID, Valid: true},
			Description: pgtype.Text{String: description, Valid: true},
		})
		if err != nil {
			return err
		}

		reversed, err = q.UpdateConversionStatus(ctx, db.UpdateConversionStatusParams{
			ID:     conversion.ID,
			Status: "reversed",
		})
		return err
	})

	return reversed, err
}
