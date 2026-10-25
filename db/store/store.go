package store

import (
	"context"
	"fmt"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store interface {
	CreateAccountTx(ctx context.Context, args CreateAccountTxArguments) (CreateAccountTxResult, error)
	FreezeAccountTx(ctx context.Context, args FreezeAccountTxParams) (int64, error)
	UnFreezeAccountTx(ctx context.Context, args UnFreezeAccountTxParams) (int64, error)
	CloseAccountTx(ctx context.Context, args CloseAccountTxParams) (int64, error)
	UpdateAccountDetailByAccountIDTx(ctx context.Context, args UpdateAccountDetailByAccountIDTxParams) (db.AccountDetail, error)
	ReplaceCardTx(ctx context.Context, args ReplaceCardTxArgs) (ReplaceCardTxResult, error)
	CreateCardTx(ctx context.Context, arg CreateCardTxParams) (db.Card, db.CardSpendingLimit, error)
	CreateConversionTransaction(ctx context.Context, args CreateConversionTransactionArgs) (CreateConversionTransactionResult, error)
	ReverseConversionTransaction(ctx context.Context, args ReverseConversionTransactionArgs) (db.Conversion, error)
	CreateTransferTransaction(ctx context.Context, args CreateTransferTransactionArgs) error
	CreateBusinessTx(ctx context.Context, args db.CreateBusinessParams, role string) (CreateBusinessTxResult, error)
	MarkInvoiceAsPaidTransaction(ctx context.Context, arg MarkInvoiceAsPaidTransactionParams) error
	CompleteIncomingPaymentTx(ctx context.Context, args CompleteIncomingPaymentTxArgs) (CompleteIncomingPaymentTxResult, error)

	db.Querier
}
type SQLStore struct {
	*db.Queries
	db *pgxpool.Pool
}

func NewStore(pool *pgxpool.Pool) Store {
	return &SQLStore{
		db:      pool,
		Queries: db.New(pool),
	}
}

func (store *SQLStore) execTx(
	ctx context.Context,
	fn func(*db.Queries) error,
) error {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := store.Queries.WithTx(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}

		return err
	}

	return tx.Commit(ctx)
}
