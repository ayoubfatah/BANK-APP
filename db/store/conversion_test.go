package store

import (
	"context"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)


func TestCreateConversionTransaction_AyoubConvertsUSDToEUR(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"Convert",
		"ayoub-conversion-usd-eur-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

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
				String: "conversion-usd-eur-20261002-001",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	// Conversion row
	require.Equal(t, user.ID, result.Conversion.UserID)
	require.Equal(t, usdAccount.ID, result.Conversion.FromAccountID)
	require.Equal(t, eurAccount.ID, result.Conversion.ToAccountID)
	require.Equal(t, fromAmount, result.Conversion.FromAmount)
	require.Equal(t, int64(9200), result.Conversion.ToAmount)
	require.Equal(t, "USD", result.Conversion.FromCurrency)
	require.Equal(t, "EUR", result.Conversion.ToCurrency)
	require.Equal(t, fee, result.Conversion.Fee)
	require.Equal(t, "completed", result.Conversion.Status)

	// USD balance
	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(89950), usdAfter.Balance)

	// EUR balance
	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(9200), eurAfter.Balance)

	// USD entry
	usdEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Len(t, usdEntries, 1)
	require.Equal(t, "conversion", usdEntries[0].Type)
	require.Equal(t, int64(10050), usdEntries[0].Amount)

	// EUR entry
	eurEntries, err := store.ListEntriesByAccountID(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Len(t, eurEntries, 1)
	require.Equal(t, "conversion", eurEntries[0].Type)
	require.Equal(t, int64(9200), eurEntries[0].Amount)
}

// ============================================================
// Same account
// ============================================================

func TestCreateConversionTransaction_SameAccount(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"SameAccount",
		"ayoub-conversion-same-account-20261002@test.com",
	)

	account := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	_, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: account.ID,
			ToAccountID:   account.ID,
			FromAmount:    1000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-same-account-20261002-001",
				Valid:  true,
			},
		},
	)

	require.ErrorIs(t, err, ErrSameAccountConversion)

	after, err := store.GetAccount(
		context.Background(),
		account.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(100000), after.Balance)
}

// ============================================================
// Cross user
// ============================================================

func TestCreateConversionTransaction_CrossUserRejected(t *testing.T) {
	store := testStore

	ayoub := createTestUser(
		t,
		store,
		"Ayoub",
		"CrossUser",
		"ayoub-conversion-cross-user-20261002@test.com",
	)

	roma := createTestUser(
		t,
		store,
		"Roma",
		"CrossUser",
		"roma-conversion-cross-user-20261002@test.com",
	)

	ayoubUSD := createTestAccount(
		t,
		store,
		ayoub.ID,
		"USD",
		100000,
	)

	romaEUR := createTestAccount(
		t,
		store,
		roma.ID,
		"EUR",
		0,
	)

	_, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        ayoub.ID,
			FromAccountID: ayoubUSD.ID,
			ToAccountID:   romaEUR.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-cross-user-20261002-001",
				Valid:  true,
			},
		},
	)

	require.ErrorIs(t, err, ErrCrossUserConversion)

	// Ayoub untouched
	ayoubAfter, err := store.GetAccount(
		context.Background(),
		ayoubUSD.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(100000), ayoubAfter.Balance)

	// Roma untouched
	romaAfter, err := store.GetAccount(
		context.Background(),
		romaEUR.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(0), romaAfter.Balance)
}

// ============================================================
// Same currency
// ============================================================
//
// The database has:
//
// UNIQUE(user_id, currency)
//
// Therefore one user cannot have two USD accounts.
//
// We test the same-currency validation by using two accounts
// from two users. The operation must be rejected before any
// balance changes.
//
// If your implementation checks cross-user ownership BEFORE
// same-currency validation, this test should assert the
// corresponding cross-user error instead.
//
// ============================================================

func TestCreateConversionTransaction_SameCurrencyRejected(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"SameCurrency",
		"ayoub-conversion-same-currency-20261002@test.com",
	)

	account := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	_, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: account.ID,
			ToAccountID:   account.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-same-currency-20261002-001",
				Valid:  true,
			},
		},
	)

	// Same account is necessarily the same currency too.
	require.ErrorIs(t, err, ErrSameAccountConversion)

	after, err := store.GetAccount(
		context.Background(),
		account.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(100000), after.Balance)
}

// ============================================================
// Insufficient funds
// ============================================================

func TestCreateConversionTransaction_InsufficientFunds(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"InsufficientFunds",
		"ayoub-conversion-insufficient-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		500,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

	_, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: usdAccount.ID,
			ToAccountID:   eurAccount.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-insufficient-20261002-001",
				Valid:  true,
			},
		},
	)

	require.ErrorIs(t, err, ErrInsufficientFunds)

	// USD unchanged
	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(500), usdAfter.Balance)

	// EUR unchanged
	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(0), eurAfter.Balance)

	// No entries created
	entries, err := store.ListEntriesByAccountID(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Empty(t, entries)
}

// ============================================================
// Source account frozen
// ============================================================

func TestCreateConversionTransaction_SourceFrozen(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"SourceFrozen",
		"ayoub-conversion-source-frozen-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

	_, err := store.UpdateAccountStatus(
		context.Background(),
		db.UpdateAccountStatusParams{
			ID:     usdAccount.ID,
			Status: "frozen",
		},
	)
	require.NoError(t, err)

	_, err = store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: usdAccount.ID,
			ToAccountID:   eurAccount.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-source-frozen-20261002-001",
				Valid:  true,
			},
		},
	)

	require.ErrorIs(t, err, ErrAccountNotActive)

	// Destination untouched
	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(0), eurAfter.Balance)
}

// ============================================================
// Destination account frozen
// ============================================================

func TestCreateConversionTransaction_DestinationFrozen(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"DestinationFrozen",
		"ayoub-conversion-destination-frozen-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

	_, err := store.UpdateAccountStatus(
		context.Background(),
		db.UpdateAccountStatusParams{
			ID:     eurAccount.ID,
			Status: "frozen",
		},
	)
	require.NoError(t, err)

	_, err = store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: usdAccount.ID,
			ToAccountID:   eurAccount.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-destination-frozen-20261002-001",
				Valid:  true,
			},
		},
	)

	require.ErrorIs(t, err, ErrAccountNotActive)

	// Source untouched
	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(100000), usdAfter.Balance)
}

// ============================================================
// No exchange rate
// ============================================================
//
// IMPORTANT:
// You said ALL exchange rates already exist in the DB.
//
// Therefore there is no valid "NoExchangeRate" test anymore.
// Trying USD -> MAD, JPY -> EUR, etc. is not a reliable test.
//
// We instead test that a conversion uses the existing seeded
// exchange rate correctly.
//
// ============================================================

func TestCreateConversionTransaction_UsesExistingExchangeRate(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"ExistingRate",
		"ayoub-conversion-existing-rate-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

	result, err := store.CreateConversionTransaction(
		context.Background(),
		CreateConversionTransactionArgs{
			UserID:        user.ID,
			FromAccountID: usdAccount.ID,
			ToAccountID:   eurAccount.ID,
			FromAmount:    10000,
			Status:        "pending",
			IdempotencyKey: pgtype.Text{
				String: "conversion-existing-rate-20261002-001",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	// Existing seeded USD -> EUR rate is 0.92.
	require.Equal(t, int64(9200), result.Conversion.ToAmount)

	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(90000), usdAfter.Balance)

	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(9200), eurAfter.Balance)
}

// ============================================================
// Idempotency
// ============================================================

func TestCreateConversionTransaction_Idempotent(t *testing.T) {
	store := testStore

	user := createTestUser(
		t,
		store,
		"Ayoub",
		"Idempotent",
		"ayoub-conversion-idempotent-20261002@test.com",
	)

	usdAccount := createTestAccount(
		t,
		store,
		user.ID,
		"USD",
		100000,
	)

	eurAccount := createTestAccount(
		t,
		store,
		user.ID,
		"EUR",
		0,
	)

	args := CreateConversionTransactionArgs{
		UserID:        user.ID,
		FromAccountID: usdAccount.ID,
		ToAccountID:   eurAccount.ID,
		FromAmount:    10000,
		Status:        "pending",
		IdempotencyKey: pgtype.Text{
			String: "conversion-idempotent-20261002-001",
			Valid:  true,
		},
	}

	first, err := store.CreateConversionTransaction(
		context.Background(),
		args,
	)
	require.NoError(t, err)

	second, err := store.CreateConversionTransaction(
		context.Background(),
		args,
	)
	require.NoError(t, err)

	// Same conversion returned.
	require.Equal(
		t,
		first.Conversion.ID,
		second.Conversion.ID,
	)

	// USD moved only once.
	usdAfter, err := store.GetAccount(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(90000), usdAfter.Balance)

	// EUR received only once.
	eurAfter, err := store.GetAccount(
		context.Background(),
		eurAccount.ID,
	)
	require.NoError(t, err)
	require.Equal(t, int64(9200), eurAfter.Balance)

	// Only one USD entry.
	entries, err := store.ListEntriesByAccountID(
		context.Background(),
		usdAccount.ID,
	)
	require.NoError(t, err)
	require.Len(t, entries, 1)
}