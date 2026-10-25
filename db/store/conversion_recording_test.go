package store

import (
	"context"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestRecording_CreateUser(t *testing.T) {
	store := testStore

	_, err := store.CreateUser(
		context.Background(),
		db.CreateUserParams{
			FirstName:    "Ayoub",
			LastName:     "Convert",
			Email:        "ayoub-conversion-recording@test.com",
			PasswordHash: "test_hashed_password",
			Status:       "active",
		},
	)

	require.NoError(t, err)
}

func TestRecording_CreateAccountsConversion(t *testing.T) {
	store := testStore

	user, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-conversion-recording@test.com",
	)
	require.NoError(t, err)

	_, err = store.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Balance:  100000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	_, err = store.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   user.ID,
			Currency: "EUR",
			Balance:  0,
			Status:   "active",
		},
	)
	require.NoError(t, err)
}

func TestRecording_ShowBalancesBeforeConversion(t *testing.T) {
	store := testStore

	user, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-conversion-recording@test.com",
	)
	require.NoError(t, err)

	usdAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   user.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	eurAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   user.ID,
			Currency: "EUR",
		},
	)
	require.NoError(t, err)

	require.Equal(t, int64(100000), usdAccount.Balance)
	require.Equal(t, int64(0), eurAccount.Balance)
}

func TestRecording_CreateConversionTransaction_AyoubConvertsUSDToEUR(t *testing.T) {
	store := testStore

	user, err := store.GetUserByEmail(
		context.Background(),
		"ayoub-conversion-recording@test.com",
	)
	require.NoError(t, err)

	usdAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   user.ID,
			Currency: "USD",
		},
	)
	require.NoError(t, err)

	eurAccount, err := store.GetAccountByUserAndCurrency(
		context.Background(),
		db.GetAccountByUserAndCurrencyParams{
			UserID:   user.ID,
			Currency: "EUR",
		},
	)
	require.NoError(t, err)

	fromAmount := int64(10000)
	fee := int64(50)

	result, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: usdAccount.ID,
			ToAccountID:   eurAccount.ID,
			FromAmount:    fromAmount,
			Fee:           fee,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-recording-usd-eur-001",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	// Verify conversion.
	require.Equal(t, user.ID, result.Conversion.UserID)
	require.Equal(t, usdAccount.ID, result.Conversion.FromAccountID)
	require.Equal(t, eurAccount.ID, result.Conversion.ToAccountID)
	require.Equal(t, fromAmount, result.Conversion.FromAmount)
	require.Equal(t, int64(9200), result.Conversion.ToAmount)
	require.Equal(t, "USD", result.Conversion.FromCurrency)
	require.Equal(t, "EUR", result.Conversion.ToCurrency)
	require.Equal(t, fee, result.Conversion.Fee)
	require.Equal(t, "completed", result.Conversion.Status)

	// Verify USD balance.
	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(89950), usdAfter.Balance)

	// Verify EUR balance.
	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)

	require.Equal(t, int64(9200), eurAfter.Balance)

	// Verify USD ledger entry.
	usdEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)

	require.Len(t, usdEntries, 1)
	require.Equal(t, "conversion", usdEntries[0].Type)
	require.Equal(t, int64(10050), usdEntries[0].Amount)

	// Verify EUR ledger entry.
	eurEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)

	require.Len(t, eurEntries, 1)
	require.Equal(t, "conversion", eurEntries[0].Type)
	require.Equal(t, int64(9200), eurEntries[0].Amount)
}
