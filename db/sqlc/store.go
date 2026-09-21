package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	*Queries
	db *pgxpool.Pool
}

func NewStore(db *pgxpool.Pool) *Store {
	return &Store{
		db:      db,
		Queries: New(db),
	}
}

func (store *Store) execTx(
	ctx context.Context,
	fn func(*Queries) error,
) error {
	tx, err := store.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	q := store.WithTx(tx)

	err = fn(q)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("tx err: %v, rb err: %v", err, rbErr)
		}

		return err
	}

	return tx.Commit(ctx)
}


// TransferTxParams contains the input parameters of the transfer transaction 
type TransferTxParams struct {
	FromAccountID int64 `json:"from_account_id"`
	ToAccountID	int64 `json:"to_account_id"`
	Amount int64 `json:"amount"`
}

type TransferTxResult struct {
	Transfer Transfer `json:"transfer"`
	FromAccount Account `json:"from_account`
	ToAccount  Account `json:"to_account`
	FromEntry Entry `json:"from_entry`
	toEntry Entry `json:"to_entry"`

}
// transferTx performs a money transfer from one account to the other 
// it creates a transfer record add account entries and update accounts' balance within a single database transaction


// func (store *Store) TransferTx(ctx context.Context , arg TransferTxParams)(TransferTxResult , error){
//   var result TransferTxResult

//   err := store.execTx(ctx , func(q *Queries) error {
// 	result.Transfer
// 	return  nil
//   })
 

//   return result , err

// }