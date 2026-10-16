package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createTestEntry(
	t *testing.T,
	accountID int64,
	amount int64,
	entryType string,
	referenceID pgtype.Int8,
) Entry {
	t.Helper()

	entry, err := testQueries.CreateEntry(context.Background(), CreateEntryParams{
		AccountID:   accountID,
		Amount:      amount,
		Currency:    "MAD",
		Type:        entryType,
		ReferenceID: referenceID,
		Description: pgtype.Text{
			String: "Test " + entryType,
			Valid:  true,
		},
	})
	require.NoError(t, err)

	return entry
}

func TestCreateEntry(t *testing.T) {
	account := createRandomAccount(t)

	entry := createTestEntry(t, account.ID, 5000, "deposit", pgtype.Int8{})

	require.NotZero(t, entry.ID)
	require.Equal(t, account.ID, entry.AccountID)
	require.Equal(t, int64(5000), entry.Amount)
	require.Equal(t, "MAD", entry.Currency)
	require.Equal(t, "deposit", entry.Type)
	require.Equal(t, "Test deposit", entry.Description.String)
	require.False(t, entry.ReferenceID.Valid)
	require.True(t, entry.CreatedAt.Valid)
}

func TestGetEntry(t *testing.T) {
	account := createRandomAccount(t)
	created := createTestEntry(t, account.ID, 5000, "deposit", pgtype.Int8{})

	entry, err := testQueries.GetEntry(context.Background(), created.ID)

	require.NoError(t, err)
	require.Equal(t, created.ID, entry.ID)
	require.Equal(t, created.AccountID, entry.AccountID)
	require.Equal(t, created.Amount, entry.Amount)
	require.Equal(t, created.Currency, entry.Currency)
	require.Equal(t, created.Type, entry.Type)
	require.Equal(t, created.Description, entry.Description)
	require.Equal(t, created.ReferenceID, entry.ReferenceID)
}

func TestListEntriesByAccountID(t *testing.T) {
	account := createRandomAccount(t)

	for i := 0; i < 5; i++ {
		createTestEntry(t, account.ID, int64((i+1)*1000), "deposit", pgtype.Int8{})
	}

	entries, err := testQueries.ListEntriesByAccountID(context.Background(), account.ID)

	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.Equal(t, account.ID, entry.AccountID)
	}
}

func TestListEntriesByAccountIDFiltered(t *testing.T) {
	account := createRandomAccount(t)

	createTestEntry(t, account.ID, 1000, "deposit", pgtype.Int8{})
	createTestEntry(t, account.ID, 5000, "deposit", pgtype.Int8{})
	createTestEntry(t, account.ID, 3000, "withdrawal", pgtype.Int8{})
	createTestEntry(t, account.ID, 500, "fee", pgtype.Int8{})

	entries, err := testQueries.ListEntriesByAccountIDFiltered(
		context.Background(),
		ListEntriesByAccountIDFilteredParams{
			AccountID: account.ID,
			FromDate:  pgtype.Timestamptz{},
			ToDate:    pgtype.Timestamptz{},
			EntryType: pgtype.Text{String: "deposit", Valid: true},
			MinAmount: pgtype.Int8{Int64: 1000, Valid: true},
			MaxAmount: pgtype.Int8{Int64: 4000, Valid: true},
			PageSize:  20,
			Offset:    0,
		},
	)

	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, int64(1000), entries[0].Amount)
	require.Equal(t, "deposit", entries[0].Type)
}

func TestListEntriesByAccountIDFilteredWithNoFilters(t *testing.T) {
	account := createRandomAccount(t)

	for i := 0; i < 3; i++ {
		createTestEntry(t, account.ID, int64((i+1)*1000), "deposit", pgtype.Int8{})
	}

	entries, err := testQueries.ListEntriesByAccountIDFiltered(
		context.Background(),
		ListEntriesByAccountIDFilteredParams{
			AccountID: account.ID,
			FromDate:  pgtype.Timestamptz{},
			ToDate:    pgtype.Timestamptz{},
			EntryType: pgtype.Text{},
			MinAmount: pgtype.Int8{},
			MaxAmount: pgtype.Int8{},
			PageSize:  20,
			Offset:    0,
		},
	)

	require.NoError(t, err)
	require.Len(t, entries, 3)
}

func TestListEntriesByAccountIDFilteredByDate(t *testing.T) {
	account := createRandomAccount(t)
	created := createTestEntry(t, account.ID, 1000, "deposit", pgtype.Int8{})

	entries, err := testQueries.ListEntriesByAccountIDFiltered(
		context.Background(),
		ListEntriesByAccountIDFilteredParams{
			AccountID: account.ID,
			FromDate: pgtype.Timestamptz{
				Time:  created.CreatedAt.Time.Add(-time.Minute),
				Valid: true,
			},
			ToDate: pgtype.Timestamptz{
				Time:  created.CreatedAt.Time.Add(time.Minute),
				Valid: true,
			},
			EntryType: pgtype.Text{},
			MinAmount: pgtype.Int8{},
			MaxAmount: pgtype.Int8{},
			PageSize:  20,
			Offset:    0,
		},
	)

	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Equal(t, created.ID, entries[0].ID)
}

func TestGetEntryByAccountID(t *testing.T) {
	account := createRandomAccount(t)
	otherAccount := createRandomAccount(t)

	created := createTestEntry(t, account.ID, 2500, "deposit", pgtype.Int8{})

	entry, err := testQueries.GetEntryByAccountID(
		context.Background(),
		GetEntryByAccountIDParams{
			ID:        created.ID,
			AccountID: account.ID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, created.ID, entry.ID)
	require.Equal(t, account.ID, entry.AccountID)

	_, err = testQueries.GetEntryByAccountID(
		context.Background(),
		GetEntryByAccountIDParams{
			ID:        created.ID,
			AccountID: otherAccount.ID,
		},
	)

	require.Error(t, err)
}

func TestGetEntriesByReferenceID(t *testing.T) {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	referenceID := pgtype.Int8{Int64: 12345, Valid: true}

	entry1 := createTestEntry(t, account1.ID, 5000, "withdrawal", referenceID)
	entry2 := createTestEntry(t, account2.ID, 5000, "deposit", referenceID)
	createTestEntry(t, account1.ID, 1000, "deposit", pgtype.Int8{})

	entries, err := testQueries.GetEntriesByReferenceID(context.Background(), referenceID)

	require.NoError(t, err)
	require.Len(t, entries, 2)

	ids := map[int64]bool{}
	for _, entry := range entries {
		require.True(t, entry.ReferenceID.Valid)
		require.Equal(t, referenceID.Int64, entry.ReferenceID.Int64)
		ids[entry.ID] = true
	}

	require.True(t, ids[entry1.ID])
	require.True(t, ids[entry2.ID])
}

func TestGetAccountBalanceFromEntries(t *testing.T) {
	account := createRandomAccount(t)

	createTestEntry(t, account.ID, 10000, "deposit", pgtype.Int8{})
	createTestEntry(t, account.ID, 3000, "withdrawal", pgtype.Int8{})
	createTestEntry(t, account.ID, 2000, "fee", pgtype.Int8{})

	balance, err := testQueries.GetAccountBalanceFromEntries(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Equal(t, int64(5000), balance)
}

func TestGetAccountBalanceFromEntriesWithoutEntries(t *testing.T) {
	account := createRandomAccount(t)

	balance, err := testQueries.GetAccountBalanceFromEntries(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Equal(t, int64(0), balance)
}

func TestGetAccountOwnerAndBalance(t *testing.T) {
	account := createRandomAccount(t)

	result, err := testQueries.GetAccountOwnerAndBalance(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Equal(t, account.ID, result.ID)
	require.Equal(t, account.UserID, result.UserID)
}
