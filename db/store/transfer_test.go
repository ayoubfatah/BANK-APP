package store

import (
	"context"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTestUser(t *testing.T, store *Store, firstName, lastName, email string) db.User {
	user, err := store.CreateUser(context.Background(), db.CreateUserParams{
		FirstName:    firstName,
		LastName:     lastName,
		Email:        email,
		PasswordHash: "test_hashed_password",
		Status:       "active",
	})
	require.NoError(t, err)
	return user
}

func createTestAccount(t *testing.T, store *Store, userID int64, currency string, balance int64) db.Account {
	account, err := store.CreateAccount(context.Background(), db.CreateAccountParams{
		UserID:   userID,
		Currency: currency,
		Balance:  balance,
		Status:   "active",
	})
	require.NoError(t, err)
	return account
}

func TestCreateTransferTransaction_AyoubSendsRoma(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Test", "ayoub@test.com")
	romaUser := createTestUser(t, store, "Roma", "Test", "roma@test.com")

	ayoubBalance := 100000
	romaBalance := 50000

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", int64(ayoubBalance)) // $1,000.00
	roma := createTestAccount(t, store, romaUser.ID, "USD", int64(romaBalance))    // $500.00

	amount := int64(50000) // $100.00

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID: ayoub.ID,
		ToAccountID:   roma.ID,
		Amount:        amount,
		Currency:      "USD",
		Status:        "pending",
		Reference:     pgtype.Text{String: "ayoub -> roma test", Valid: true},
		IdempotencyKey: pgtype.Text{
			String: "test-transfer-ayoub-roma-1",
			Valid:  true,
		},
	})
	require.NoError(t, err)

	// balances actually moved
	ayoubAfter, err := store.GetAccount(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Equal(t, int64(ayoubBalance-int(amount)), ayoubAfter.Balance)

	romaAfter, err := store.GetAccount(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Equal(t, int64(romaBalance+int(amount)), romaAfter.Balance)

	// entries written correctly
	ayoubEntries, err := store.ListEntriesByAccountID(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Len(t, ayoubEntries, 1)
	require.Equal(t, "withdrawal", ayoubEntries[0].Type)
	require.Equal(t, amount, ayoubEntries[0].Amount)

	romaEntries, err := store.ListEntriesByAccountID(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Len(t, romaEntries, 1)
	require.Equal(t, "deposit", romaEntries[0].Type)
	require.Equal(t, amount, romaEntries[0].Amount)
}

func TestCreateTransferTransaction_SameAccount(t *testing.T) {
	store := testStore

	user := createTestUser(t, store, "Ayoub", "Test", "ayoub-same@test.com")
	account := createTestAccount(t, store, user.ID, "USD", 100000)

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  account.ID,
		ToAccountID:    account.ID,
		Amount:         1000,
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-same-account", Valid: true},
	})

	require.ErrorIs(t, err, ErrSameAccount)

	// nothing should have changed
	after, err := store.GetAccount(context.Background(), account.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), after.Balance)
}

func TestCreateTransferTransaction_InsufficientFunds(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Poor", "ayoub-poor@test.com")
	romaUser := createTestUser(t, store, "Roma", "Rich", "roma-rich@test.com")

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 500) // only $5.00
	roma := createTestAccount(t, store, romaUser.ID, "USD", 100000)

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    roma.ID,
		Amount:         10000, // trying to send $100.00
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-insufficient-funds", Valid: true},
	})

	require.ErrorIs(t, err, ErrInsufficientFunds)

	// rollback check: neither balance moved, no entries written
	ayoubAfter, err := store.GetAccount(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Equal(t, int64(500), ayoubAfter.Balance)

	romaAfter, err := store.GetAccount(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), romaAfter.Balance)

	ayoubEntries, err := store.ListEntriesByAccountID(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Empty(t, ayoubEntries)

	romaEntries, err := store.ListEntriesByAccountID(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Empty(t, romaEntries)
}

func TestCreateTransferTransaction_CurrencyMismatch(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "USD", "ayoub-usd@test.com")
	romaUser := createTestUser(t, store, "Roma", "EUR", "roma-eur@test.com")

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 100000)
	roma := createTestAccount(t, store, romaUser.ID, "EUR", 100000)

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    roma.ID,
		Amount:         10000,
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-currency-mismatch", Valid: true},
	})

	require.ErrorIs(t, err, ErrCurrencyMismatch)

	ayoubAfter, err := store.GetAccount(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), ayoubAfter.Balance)

	romaAfter, err := store.GetAccount(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), romaAfter.Balance)
}

func TestCreateTransferTransaction_SenderFrozen(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Frozen", "ayoub-frozen@test.com")
	romaUser := createTestUser(t, store, "Roma", "Test", "roma-recv@test.com")

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 100000)
	roma := createTestAccount(t, store, romaUser.ID, "USD", 100000)

	_, err := store.UpdateAccountStatus(context.Background(), db.UpdateAccountStatusParams{
		ID:     ayoub.ID,
		Status: "frozen",
	})
	require.NoError(t, err)

	err = store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    roma.ID,
		Amount:         10000,
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-sender-frozen", Valid: true},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")

	romaAfter, err := store.GetAccount(context.Background(), roma.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), romaAfter.Balance) // untouched
}

func TestCreateTransferTransaction_ReceiverFrozen(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Test", "ayoub-send2@test.com")
	romaUser := createTestUser(t, store, "Roma", "Frozen", "roma-frozen@test.com")

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 100000)
	roma := createTestAccount(t, store, romaUser.ID, "USD", 100000)

	_, err := store.UpdateAccountStatus(context.Background(), db.UpdateAccountStatusParams{
		ID:     roma.ID,
		Status: "frozen",
	})
	require.NoError(t, err)

	err = store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    roma.ID,
		Amount:         10000,
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-receiver-frozen", Valid: true},
	})

	require.Error(t, err)
	require.Contains(t, err.Error(), "not active")

	ayoubAfter, err := store.GetAccount(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), ayoubAfter.Balance) // untouched — rollback
}

func TestCreateTransferTransaction_ReceiverDoesNotExist(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Test", "ayoub-noreceiver@test.com")
	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 100000)

	const nonExistentAccountID = 999999999

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    nonExistentAccountID,
		Amount:         10000,
		Currency:       "USD",
		Status:         "pending",
		IdempotencyKey: pgtype.Text{String: "test-receiver-missing", Valid: true},
	})

	require.Error(t, err)

	ayoubAfter, err := store.GetAccount(context.Background(), ayoub.ID)
	require.NoError(t, err)
	require.Equal(t, int64(100000), ayoubAfter.Balance) // rollback: nothing deducted
}

func TestCreateTransferTransaction_StatusIsCompleted(t *testing.T) {
	store := testStore

	ayoubUser := createTestUser(t, store, "Ayoub", "Status", "ayoub-status@test.com")
	romaUser := createTestUser(t, store, "Roma", "Status", "roma-status@test.com")

	ayoub := createTestAccount(t, store, ayoubUser.ID, "USD", 100000)
	roma := createTestAccount(t, store, romaUser.ID, "USD", 100000)

	err := store.CreateTransferTransaction(context.Background(), CreateTransferTransactionArgs{
		FromAccountID:  ayoub.ID,
		ToAccountID:    roma.ID,
		Amount:         5000,
		Currency:       "USD",
		Status:         "pending", // starts pending, function should flip it
		IdempotencyKey: pgtype.Text{String: "test-status-completed", Valid: true},
	})
	require.NoError(t, err)

	transfer, err := store.GetTransferByIdempotencyKey(context.Background(), pgtype.Text{
		String: "test-status-completed", Valid: true,
	})
	require.NoError(t, err)
	require.Equal(t, "completed", transfer.Status)
}
