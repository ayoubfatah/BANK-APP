package store

import (
	"context"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestRecording_CreateUsers(t *testing.T) {
	store := testStore

	_, err := store.CreateUser(context.Background(), db.CreateUserParams{
		FirstName:    "Ayoub",
		LastName:     "Fatah",
		Email:        "ayoub-recording@test.com",
		PasswordHash: "test_hashed_password",
		Status:       "active",
	})
	require.NoError(t, err)

	_, err = store.CreateUser(context.Background(), db.CreateUserParams{
		FirstName:    "Youssef",
		LastName:     "Test",
		Email:        "youssef-recording@test.com",
		PasswordHash: "test_hashed_password",
		Status:       "active",
	})
	require.NoError(t, err)
}

func TestRecording_CreateAccounts(t *testing.T) {
	store := NewStore(testStore.db)

	ayoub, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-recording@test.com",
	)
	require.NoError(t, err)

	youssef, err := store.GetUserByEmail(
		context.Background(),
		"youssef-recording@test.com",
	)
	require.NoError(t, err)

	_, err = store.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   ayoub.ID,
			Currency: "USD",
			Balance:  100000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	_, err = store.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   youssef.ID,
			Currency: "USD",
			Balance:  50000,
			Status:   "active",
		},
	)
	require.NoError(t, err)
}

func TestRecording_ShowBalancesBeforeTransfer(t *testing.T) {
	store := testStore

	ayoub, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-recording@test.com",
	)
	require.NoError(t, err)

	youssef, err := store.GetUserByEmail(
		context.Background(),
		"youssef-recording@test.com",
	)
	require.NoError(t, err)

	ayoubAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   ayoub.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	youssefAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   youssef.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	require.Equal(t, int64(100000), ayoubAccount.Balance)
	require.Equal(t, int64(50000), youssefAccount.Balance)
}

func TestCreateTransferTransaction_AyoubSendsYoussef(t *testing.T) {
	store := testStore

	ayoub, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-recording@test.com",
	)
	require.NoError(t, err)

	youssef, err := store.GetUserByEmail(
		context.Background(),
		"youssef-recording@test.com",
	)
	require.NoError(t, err)

	ayoubAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   ayoub.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	youssefAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   youssef.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	amount := int64(10000)

	err = store.CreateTransferTransaction(
		context.Background(),
		CreateTransferTransactionArgs{
			FromAccountID: ayoubAccount.ID,
			ToAccountID:   youssefAccount.ID,
			Amount:        amount,
			Currency:      "USD",
			Status:        "pending",
			Reference: pgtype.Text{
				String: "Transfer to account 2",
				Valid:  true,
			},
			IdempotencyKey: pgtype.Text{
				String: "recording-transfer-ayoub-youssef",
				Valid:  true,
			},
		},
	)
	require.NoError(t, err)

	ayoubAfter, err := store.GetAccount(
		context.Background(),
		ayoubAccount.ID,
	)
	require.NoError(t, err)

	youssefAfter, err := store.GetAccount(
		context.Background(),
		youssefAccount.ID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(90000), ayoubAfter.Balance)
	require.Equal(t, int64(60000), youssefAfter.Balance)

	ayoubEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		ayoubAccount.ID,
	)
	require.NoError(t, err)

	require.Len(t, ayoubEntries, 1)
	require.Equal(t, "withdrawal", ayoubEntries[0].Type)
	require.Equal(t, amount, ayoubEntries[0].Amount)

	youssefEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		youssefAccount.ID,
	)
	require.NoError(t, err)

	require.Len(t, youssefEntries, 1)
	require.Equal(t, "deposit", youssefEntries[0].Type)
	require.Equal(t, amount, youssefEntries[0].Amount)
}
