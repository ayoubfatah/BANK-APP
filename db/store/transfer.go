package store

import (
	"context"
	"errors"
	"fmt"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ayoub sends money to roma
// check if roma exists and if ayoub exists
// check if ayoub is actually not sending money to aayoub
// check if ayoub has enough balance in currency account
// lock both accounts with lower id method
//  send money to roma
//  increase roma account +100$
// decrease ayoub account -100$
// add entry for roma as a deposit `+100 "transfered by ayoub + date ..."
// add entry for ayoub as withdrawl - 100 sent money to roma + date
// make a fee  table if fees where used


var (
	ErrSameAccount       = errors.New("cannot transfer to the same account")
	ErrAccountNotFound   = errors.New("account not found")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrCurrencyMismatch  = errors.New("accounts must use the same currency")

)

type CreateTransferTransactionArgs struct {
	FromAccountID  int64  				`json:"from_account_id"`
	ToAccountID  int64  				`json:"to_account_id"`
	Amount         int64       `json:"amount"`
	Currency       string      `json:"currency"`
	Status         string      `json:"status"`
	Reference      pgtype.Text `json:"reference"`
	IdempotencyKey pgtype.Text `json:"idempotency_key"`
}

func (store  *Store )CreateTransferTransaction(ctx context.Context , args CreateTransferTransactionArgs) error{
	// lets lock accounts first in a consistent order
    err := store.execTx(ctx , func(q *db.Queries) error {

	if args.IdempotencyKey.Valid {
    _, err := q.GetTransferByIdempotencyKey(ctx, args.IdempotencyKey)
    if err == nil {
        return nil
    }
    if !errors.Is(err, pgx.ErrNoRows) {
        return err
    }
}

	if ( args.FromAccountID == args.ToAccountID){
	return ErrSameAccount
	}

	firstID , secondID := args.FromAccountID , args.ToAccountID
	if firstID > secondID {
		firstID , secondID = secondID, firstID
	}
	first, err := q.GetAccount(ctx, firstID)
		if err != nil {
			return err
		}

		second, err := q.GetAccount(ctx, secondID)
		if err != nil {
			return err
	}
	fromAccount , toAccount  := first, second

	// lets find out which one is sending which on is receiving  the money
	    if first.ID != args.FromAccountID {
        fromAccount, toAccount = second, first
    }

	if fromAccount.Status != "active" {
			return fmt.Errorf("source account %d is not active", fromAccount.ID)
		}

	if toAccount.Status != "active" {
			return fmt.Errorf("destination account %d is not active", toAccount.ID)
	}


	// check if both profile have the same currency 
	if (fromAccount.Currency != toAccount.Currency){
		return ErrCurrencyMismatch
	}
	
	if (fromAccount.Balance < args.Amount){
		return ErrInsufficientFunds
	}

	transfer , err := q.CreateTransfer(context.Background(),

    db.CreateTransferParams{
    FromAccountID: pgtype.Int8{
        Int64: fromAccount.ID,
        Valid: true,
    },
    ToAccountID: pgtype.Int8{
        Int64: toAccount.ID,
        Valid: true,
    },
	Amount: args.Amount,
	Currency: args.Currency,
	Status: args.Status,
	Reference: args.Reference,
	IdempotencyKey: args.IdempotencyKey,
},)


	if err !=nil {
		return  err
	}
	_,err = q.UpdateTransferStatus(context.Background(),
	db.UpdateTransferStatusParams{
	ID: transfer.ID,
	Column2: "completed",
	
	})

	if err !=nil {
		return  err
	}


	//  remove money from sender 

	   _, err = q.SubtractFromAccountBalance(
            ctx,
            db.SubtractFromAccountBalanceParams{
                Balance: args.Amount,
                ID:      fromAccount.ID,
            },
        )

        if err != nil {
            return err
        }

		// add money to receiver 
	   _, err = q.AddToAccountBalance(
            ctx,
            db.AddToAccountBalanceParams{
                Balance: args.Amount,
                ID:      toAccount.ID,
            },
        )
		 if err != nil {
            return err
        }

	//  sender entry 
	fromDescription := fmt.Sprintf("Transfer to account %d", toAccount.ID)

        _, err = q.CreateEntry(
            ctx,
            db.CreateEntryParams{
                AccountID: fromAccount.ID,
                Amount:    args.Amount,
                Currency:  args.Currency,
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

		//    receiver entry 
		toDescription := fmt.Sprintf("Transfer from account %d", fromAccount.ID)

        _, err = q.CreateEntry(
            ctx,
            db.CreateEntryParams{
                AccountID: toAccount.ID,
                Amount:    args.Amount,
                Currency:  args.Currency,
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



	return nil 
	})
	return err
}