package store

import (
	"context"
	"fmt"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomAccout(t *testing.T) db.Account {
	t.Helper()

	user := createNewUser(t)

	account, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   user.ID,
			Balance:  2000,
			Status:   "active",
			Currency: "USD",
		},
	)

	require.NoError(t, err)
	require.NotZero(t, account.ID)

	return account
}

func createRandomIncomingPayment(
	t *testing.T,
	accountID int64,
	amount int64,
	externalReference string,
) db.IncomingPayment {
	t.Helper()

	payment, err := testStore.CreateIncomingPayment(
		context.Background(),
		db.CreateIncomingPaymentParams{
			AccountID: accountID,

			SenderName: pgtype.Text{
				String: fmt.Sprintf("Sender %d", accountID),
				Valid:  true,
			},

			SenderReference: pgtype.Text{
				String: "Payment for services",
				Valid:  true,
			},

			Amount:   amount,
			Currency: "USD",
			Status:   "pending",

			ExternalReference: pgtype.Text{
				String: externalReference,
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)
	require.NotZero(t, payment.ID)

	return payment
}

func TestCreateIncomingPayment(t *testing.T) {
	store := testStore

	// Create a user
	user := createNewUser(t)

	// Create the account that will receive the payment
	account, err := store.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   user.ID,
			Balance:  2000,
			Status:   "active",
			Currency: "USD",
		},
	)

	require.NoError(t, err)
	require.NotZero(t, account.ID)

	// Create the incoming payment
	incomingPayment, err := store.CreateIncomingPayment(
		context.Background(),
		db.CreateIncomingPaymentParams{
			AccountID: account.ID,

			SenderName: pgtype.Text{
				String: user.FirstName + " " + user.LastName,
				Valid:  true,
			},

			SenderReference: pgtype.Text{
				String: "Payment for services",
				Valid:  true,
			},

			Amount:   100000,
			Currency: "USD",
			Status:   "pending",

			ExternalReference: pgtype.Text{
				String: "BANK-TXN-928374",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	// Verify the generated ID
	require.NotZero(t, incomingPayment.ID)

	// Verify account
	require.Equal(t, account.ID, incomingPayment.AccountID)

	// Verify sender
	require.Equal(
		t,
		user.FirstName+" "+user.LastName,
		incomingPayment.SenderName.String,
	)
	require.True(t, incomingPayment.SenderName.Valid)

	// Verify sender reference
	require.Equal(
		t,
		"Payment for services",
		incomingPayment.SenderReference.String,
	)
	require.True(t, incomingPayment.SenderReference.Valid)

	// Verify amount
	require.Equal(t, int64(100000), incomingPayment.Amount)

	// Verify currency
	require.Equal(t, "USD", incomingPayment.Currency)

	// Verify status
	require.Equal(t, "pending", incomingPayment.Status)

	// Verify external reference
	require.Equal(
		t,
		"BANK-TXN-928374",
		incomingPayment.ExternalReference.String,
	)
	require.True(t, incomingPayment.ExternalReference.Valid)

	// Verify timestamps
	require.NotZero(t, incomingPayment.CreatedAt)
	require.False(t, incomingPayment.CompletedAt.Valid)
}

func TestCompletedIncomingPaymentTx(t *testing.T) {
	// Create a random account
	account := createRandomAccout(t)

	// Define the payment amount
	amount := int64(5000)

	externalReference := "BANK-TXN-1"

	// Create a random incoming payment
	incomingPayment := createRandomIncomingPayment(
		t,
		account.ID,
		amount,
		externalReference,
	)

	arg := CompleteIncomingPaymentTxArgs{
		ID:        incomingPayment.ID,
		AccountID: account.ID,
		Amount:    amount,
		Currency:  "USD",
		Type:      "incoming_payment",

		ReferenceID: pgtype.Int8{
			Int64: incomingPayment.ID,
			Valid: true,
		},

		Description: pgtype.Text{
			String: "Incoming payment 1",
			Valid:  true,
		},
	}

	// Complete the payment
	result, err := testStore.CompleteIncomingPaymentTx(
		context.Background(),
		arg,
	)
	require.NoError(t, err)

	// Verify incoming payment
	require.Equal(t, incomingPayment.ID, result.IncomingPayment.ID)
	require.Equal(t, "completed", result.IncomingPayment.Status)

	// Verify entry
	require.Equal(t, account.ID, result.Entry.AccountID)
	require.Equal(t, amount, result.Entry.Amount)
	require.Equal(t, "USD", result.Entry.Currency)
	require.Equal(t, "incoming_payment", result.Entry.Type)

	// Verify account
	require.Equal(t, account.ID, result.Account.ID)
	require.Equal(t, int64(7000), result.Account.Balance)

	// Verify the payment directly from the database
	updatedPayment, err := testStore.GetIncomingPayment(
		context.Background(),
		incomingPayment.ID,
	)
	require.NoError(t, err)
	require.Equal(t, "completed", updatedPayment.Status)

	// Verify the account directly from the database
	updatedAccount, err := testStore.GetAccount(
		context.Background(),
		account.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(7000), updatedAccount.Balance)
}
