package store

import (
	"context"
	"testing"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createNewBusiness(t *testing.T) db.Business {
	user := createNewUser(t)

	business, err := testStore.CreateBusiness(
		context.Background(),
		db.CreateBusinessParams{
			OwnerID: user.ID,
			Name:    utils.RandomString(10),
			RegistrationNumber: pgtype.Text{
				String: utils.RandomString(10),
				Valid:  true,
			},
			Country: "MA",
			Status:  "active",
		},
	)

	require.NoError(t, err)
	return business
}

func createInvoiceUtil(t *testing.T) db.Invoice {
	business := createNewBusiness(t)

	args := db.CreateInvoiceParams{
		BusinessID:   business.ID,
		CustomerName: "Ayoub",
		CustomerEmail: pgtype.Text{
			String: "ayoub@gmail.com",
			Valid:  true,
		},
		Amount:   20000,
		Currency: "USD",
		Status:   "draft",
		DueDate: pgtype.Date{
			Time:  time.Date(2026, 10, 28, 0, 0, 0, 0, time.UTC),
			Valid: true,
		},
		PaidAt: pgtype.Timestamptz{
			Valid: false,
		},
	}

	invoice, err := testStore.CreateInvoice(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotZero(t, invoice.ID)
	require.NotZero(t, invoice.CreatedAt)

	require.Equal(t, args.BusinessID, invoice.BusinessID)
	require.Equal(t, args.CustomerName, invoice.CustomerName)
	require.Equal(t, args.CustomerEmail, invoice.CustomerEmail)
	require.Equal(t, args.Amount, invoice.Amount)
	require.Equal(t, args.Currency, invoice.Currency)
	require.Equal(t, args.DueDate.Time, invoice.DueDate.Time)
	require.Equal(t, args.PaidAt, invoice.PaidAt)

	invoiceFromDB, err := testStore.GetInvoiceById(
		context.Background(),
		invoice.ID,
	)

	require.NoError(t, err)

	require.Equal(t, invoice.BusinessID, invoiceFromDB.BusinessID)
	require.Equal(t, invoice.CustomerName, invoiceFromDB.CustomerName)
	require.Equal(t, invoice.CustomerEmail, invoiceFromDB.CustomerEmail)
	require.Equal(t, invoice.Amount, invoiceFromDB.Amount)
	require.Equal(t, invoice.Currency, invoiceFromDB.Currency)
	require.Equal(t, invoice.DueDate.Time, invoiceFromDB.DueDate.Time)
	require.Equal(t, invoice.PaidAt, invoiceFromDB.PaidAt)

	return invoice
}

func TestMarkInvoiceAsPaidTransaction(t *testing.T) {
	fromUser := createNewUser(t)
	toUser := createNewUser(t)

	fromAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   fromUser.ID,
			Currency: "USD",
			Balance:  100000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	toAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   toUser.ID,
			Currency: "USD",
			Balance:  50000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	invoice := createInvoiceUtil(t)

	require.Equal(t, "draft", invoice.Status)

	invoice, err = testStore.IssueInvoice(
		context.Background(),
		invoice.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "pending", invoice.Status)

	err = testStore.MarkInvoiceAsPaidTransaction(
		context.Background(),
		MarkInvoiceAsPaidTransactionParams{
			InvoiceID:     invoice.ID,
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Currency:      "USD",
			Amount:        20000,
			Description:   "Invoice payment",
		},
	)
	require.NoError(t, err)

	fromAccount, err = testStore.GetAccount(
		context.Background(),
		fromAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(80000), fromAccount.Balance)

	toAccount, err = testStore.GetAccount(
		context.Background(),
		toAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(70000), toAccount.Balance)

	updatedInvoice, err := testStore.GetInvoiceById(
		context.Background(),
		invoice.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "paid", updatedInvoice.Status)
	require.True(t, updatedInvoice.PaidAt.Valid)
}

func TestMarkInvoiceAsPaidTransactionInsufficientFunds(t *testing.T) {
	fromUser := createNewUser(t)
	toUser := createNewUser(t)
	fromAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   fromUser.ID,
			Currency: "USD",
			Balance:  10000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	toAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   toUser.ID,
			Currency: "USD",
			Balance:  50000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	invoice := createInvoiceUtil(t)

	invoice, err = testStore.IssueInvoice(
		context.Background(),
		invoice.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "pending", invoice.Status)

	err = testStore.MarkInvoiceAsPaidTransaction(
		context.Background(),
		MarkInvoiceAsPaidTransactionParams{
			InvoiceID:     invoice.ID,
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Currency:      "USD",
			Amount:        20000,
			Description:   "Invoice payment",
		},
	)

	require.Error(t, err)
	require.EqualError(t, err, "insufficient funds")

	fromAccount, err = testStore.GetAccount(
		context.Background(),
		fromAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(10000), fromAccount.Balance)

	toAccount, err = testStore.GetAccount(
		context.Background(),
		toAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(50000), toAccount.Balance)

	updatedInvoice, err := testStore.GetInvoiceById(
		context.Background(),
		invoice.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "pending", updatedInvoice.Status)
	require.False(t, updatedInvoice.PaidAt.Valid)
}

func TestMarkInvoiceAsPaidTransactionInvalidInvoice(t *testing.T) {

	fromUser := createNewUser(t)
	toUser := createNewUser(t)

	fromAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   fromUser.ID,
			Currency: "USD",
			Balance:  100000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	toAccount, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   toUser.ID,
			Currency: "USD",
			Balance:  50000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	err = testStore.MarkInvoiceAsPaidTransaction(
		context.Background(),
		MarkInvoiceAsPaidTransactionParams{
			InvoiceID:     999999,
			FromAccountID: fromAccount.ID,
			ToAccountID:   toAccount.ID,
			Currency:      "USD",
			Amount:        20000,
			Description:   "Invoice payment",
		},
	)

	require.Error(t, err)

	fromAccount, err = testStore.GetAccount(
		context.Background(),
		fromAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(100000), fromAccount.Balance)

	toAccount, err = testStore.GetAccount(
		context.Background(),
		toAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(50000), toAccount.Balance)
}
