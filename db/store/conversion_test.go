package store

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

// seedExchangeRate inserts a rate directly via the pool, since exchange_rates
// has no natural "create" flow in the app itself — rates are seeded/fetched
// by a job, not created per-test through app logic.

func TestCreateConversionTransaction_AyoubConvertsUSDToEUR(t *testing.T) {
	store := testStore

	user := createTestUser(t, store, "Ayoub", "Convert", "ayoub-convert1@test.com")

	usdAccount := createTestAccount(t, store, user.ID, "USD", 100000) // $1,000.00
	eurAccount := createTestAccount(t, store, user.ID, "EUR", 0)


	fromAmount := int64(10000) // $100.00
	fee := int64(50)           // $0.50

	result, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
		UserID:        user.ID,
		FromAccountID: usdAccount.ID,
		ToAccountID:   eurAccount.ID,
		FromAmount:    fromAmount,
		Fee:           fee,
		Status:        "pending",
		IdempotencyKey: pgtype.Text{
			String: "test-conversion-ayoub-1",
			Valid:  true,
		},
	})
	require.NoError(t, err)

	// conversion row recorded correctly
	require.Equal(t, user.ID, result.Conversion.UserID)
	require.Equal(t, usdAccount.ID, result.Conversion.FromAccountID)
	require.Equal(t, eurAccount.ID, result.Conversion.ToAccountID)
	require.Equal(t, fromAmount, result.Conversion.FromAmount)
	require.Equal(t, int64(9200), result.Conversion.ToAmount) // 10000 * 0.92
	require.Equal(t, "USD", result.Conversion.FromCurrency)
	require.Equal(t, "EUR", result.Conversion.ToCurrency)
	require.Equal(t, fee, result.Conversion.Fee)
	require.Equal(t, "completed", result.Conversion.Status)

	// balances moved correctly — sender pays amount + fee, receiver gets converted amount
	usdAfter, err := store.GetAccount(context.Background(), usdAccount.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000-10000-50), usdAfter.Balance) // 89950

	eurAfter, err := store.GetAccount(context.Background(), eurAccount.ID)
	require.NoError(t, err)
	require.Equal(t, int64(9200), eurAfter.Balance)

	// entries written on both sides
	usdEntries, err := store.ListEntriesByAccountID(context.Background(), usdAccount.ID)
	require.NoError(t, err)
	require.Len(t, usdEntries, 1)
	require.Equal(t, "conversion", usdEntries[0].Type)
	require.Equal(t, int64(10050), usdEntries[0].Amount) // amount + fee

	eurEntries, err := store.ListEntriesByAccountID(context.Background(), eurAccount.ID)
	require.NoError(t, err)
	require.Len(t, eurEntries, 1)
	require.Equal(t, "conversion", eurEntries[0].Type)
	require.Equal(t, int64(9200), eurEntries[0].Amount)
}

// func TestCreateConversionTransaction_SameAccount(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "Same", "ayoub-convert-same@test.com")
// 	account := createTestAccount(t, store, user.ID, "USD", 100000)

// 	_, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: account.ID,
// 		ToAccountID:   account.ID,
// 		FromAmount:    1000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-same-account", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrSameAccountConversion)

// 	after, err := store.GetAccount(context.Background(), account.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(100000), after.Balance)
// }

// func TestCreateConversionTransaction_CrossUserRejected(t *testing.T) {
// 	store := testStore

// 	ayoub := createTestUser(t, store, "Ayoub", "CrossUser", "ayoub-cross@test.com")
// 	roma := createTestUser(t, store, "Roma", "CrossUser", "roma-cross@test.com")

// 	ayoubUSD := createTestAccount(t, store, ayoub.ID, "USD", 100000)
// 	romaEUR := createTestAccount(t, store, roma.ID, "EUR", 0)

// 	seedExchangeRate(t, store, "USD", "EUR", 0.92)

// 	_, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        ayoub.ID,
// 		FromAccountID: ayoubUSD.ID,
// 		ToAccountID:   romaEUR.ID, // belongs to a DIFFERENT user
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-cross-user", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrCrossUserConversion)

// 	ayoubAfter, err := store.GetAccount(context.Background(), ayoubUSD.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(100000), ayoubAfter.Balance) // untouched — rollback

// 	romaAfter, err := store.GetAccount(context.Background(), romaEUR.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(0), romaAfter.Balance)
// }

// func TestCreateConversionTransaction_SameCurrencyRejected(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "SameCurrency", "ayoub-samecurrency@test.com")
// 	usd1 := createTestAccount(t, store, user.ID, "USD", 100000)
// 	usd2 := createTestAccount(t, store, user.ID, "USD", 0)

// 	_, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usd1.ID,
// 		ToAccountID:   usd2.ID,
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-same-currency", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrSameCurrencyConversion)
// }

// func TestCreateConversionTransaction_InsufficientFunds(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "Poor", "ayoub-convert-poor@test.com")
// 	usdAccount := createTestAccount(t, store, user.ID, "USD", 500) // $5.00
// 	eurAccount := createTestAccount(t, store, user.ID, "EUR", 0)

// 	seedExchangeRate(t, store, "USD", "EUR", 0.92)

// 	_, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usdAccount.ID,
// 		ToAccountID:   eurAccount.ID,
// 		FromAmount:    10000, // trying to convert $100 with only $5
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-insufficient", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrInsufficientFunds)

// 	// rollback check
// 	usdAfter, err := store.GetAccount(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(500), usdAfter.Balance)

// 	eurAfter, err := store.GetAccount(context.Background(), eurAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(0), eurAfter.Balance)

// 	entries, err := store.ListEntriesByAccountID(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Empty(t, entries)
// }

// func TestCreateConversionTransaction_SourceFrozen(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "Frozen", "ayoub-convert-frozen@test.com")
// 	usdAccount := createTestAccount(t, store, user.ID, "USD", 100000)
// 	eurAccount := createTestAccount(t, store, user.ID, "EUR", 0)

// 	seedExchangeRate(t, store, "USD", "EUR", 0.92)

// 	_, err := store.UpdateAccountStatus(context.Background(), db.UpdateAccountStatusParams{
// 		ID:     usdAccount.ID,
// 		Status: "frozen",
// 	})
// 	require.NoError(t, err)

// 	_, err = store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usdAccount.ID,
// 		ToAccountID:   eurAccount.ID,
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-source-frozen", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrAccountNotActive)

// 	eurAfter, err := store.GetAccount(context.Background(), eurAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(0), eurAfter.Balance) // untouched — rollback
// }

// func TestCreateConversionTransaction_DestinationFrozen(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "DestFrozen", "ayoub-convert-destfrozen@test.com")
// 	usdAccount := createTestAccount(t, store, user.ID, "USD", 100000)
// 	eurAccount := createTestAccount(t, store, user.ID, "EUR", 0)

// 	seedExchangeRate(t, store, "USD", "EUR", 0.92)

// 	_, err := store.UpdateAccountStatus(context.Background(), db.UpdateAccountStatusParams{
// 		ID:     eurAccount.ID,
// 		Status: "frozen",
// 	})
// 	require.NoError(t, err)

// 	_, err = store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usdAccount.ID,
// 		ToAccountID:   eurAccount.ID,
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-dest-frozen", Valid: true},
// 	})

// 	require.ErrorIs(t, err, ErrAccountNotActive)

// 	usdAfter, err := store.GetAccount(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(100000), usdAfter.Balance) // untouched — rollback
// }

// func TestCreateConversionTransaction_NoExchangeRate(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "NoRate", "ayoub-norate@test.com")
// 	usdAccount := createTestAccount(t, store, user.ID, "USD", 100000)
// 	// use an uncommon pair we never seed, e.g. USD -> a currency with no rate row
// 	madAccount := createTestAccount(t, store, user.ID, "MAD", 0)

// 	// deliberately NOT seeding USD -> MAD for this test

// 	_, err := store.CreateConversionTransaction(context.Background(), CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usdAccount.ID,
// 		ToAccountID:   madAccount.ID,
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-no-rate", Valid: true},
// 	})

// 	require.Error(t, err)

// 	usdAfter, err := store.GetAccount(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(100000), usdAfter.Balance) // rollback: nothing moved
// }

// func TestCreateConversionTransaction_Idempotent(t *testing.T) {
// 	store := testStore

// 	user := createTestUser(t, store, "Ayoub", "Idempotent", "ayoub-convert-idem@test.com")
// 	usdAccount := createTestAccount(t, store, user.ID, "USD", 100000)
// 	eurAccount := createTestAccount(t, store, user.ID, "EUR", 0)

// 	seedExchangeRate(t, store, "USD", "EUR", 0.92)

// 	args := CreateConversionTransactionArgs{
// 		UserID:        user.ID,
// 		FromAccountID: usdAccount.ID,
// 		ToAccountID:   eurAccount.ID,
// 		FromAmount:    10000,
// 		Status:        "pending",
// 		IdempotencyKey: pgtype.Text{String: "test-conversion-idempotent-key", Valid: true},
// 	}

// 	first, err := store.CreateConversionTransaction(context.Background(), args)
// 	require.NoError(t, err)

// 	second, err := store.CreateConversionTransaction(context.Background(), args)
// 	require.NoError(t, err)

// 	// same conversion returned, not a new one
// 	require.Equal(t, first.Conversion.ID, second.Conversion.ID)

// 	// balance only moved ONCE, not twice
// 	usdAfter, err := store.GetAccount(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(100000-10000), usdAfter.Balance)

// 	eurAfter, err := store.GetAccount(context.Background(), eurAccount.ID)
// 	require.NoError(t, err)
// 	require.Equal(t, int64(9200), eurAfter.Balance)

// 	entries, err := store.ListEntriesByAccountID(context.Background(), usdAccount.ID)
// 	require.NoError(t, err)
// 	require.Len(t, entries, 1) // not 2
// }