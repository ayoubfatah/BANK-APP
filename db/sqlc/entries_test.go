package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateEntry(t *testing.T) {
	account := createRandomAccount(t)

	arg := CreateEntryParams{
		AccountID: account.ID,
		Amount:    5000,
		Currency:  "MAD",
		Type:      "deposit",
		Description: pgtype.Text{
			String: "Initial deposit",
			Valid:  true,
		},
	}

	entry, err := testQueries.CreateEntry(context.Background(), arg)

	require.NoError(t, err)
	require.NotZero(t, entry.ID)
	require.Equal(t, account.ID, entry.AccountID)
	require.Equal(t, int64(5000), entry.Amount)
	require.Equal(t, "MAD", entry.Currency)
	require.Equal(t, "deposit", entry.Type)
	require.Equal(t, "Initial deposit", entry.Description.String)
	require.NotZero(t, entry.CreatedAt)
}

func TestGetEntry(t *testing.T) {
	account := createRandomAccount(t)

	arg := CreateEntryParams{
		AccountID: account.ID,
		Amount:    5000,
		Currency:  "MAD",
		Type:      "deposit",
		Description: pgtype.Text{
			String: "Initial deposit",
			Valid:  true,
		},
	}

	createdEntry, err := testQueries.CreateEntry(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	entry, err := testQueries.GetEntry(
		context.Background(),
		createdEntry.ID,
	)

	require.NoError(t, err)

	require.Equal(t, createdEntry.ID, entry.ID)
	require.Equal(t, createdEntry.AccountID, entry.AccountID)
	require.Equal(t, createdEntry.Amount, entry.Amount)
	require.Equal(t, createdEntry.Currency, entry.Currency)
	require.Equal(t, createdEntry.Type, entry.Type)
	require.Equal(t, createdEntry.Description, entry.Description)
}

func TestListEntriesByAccountID(t *testing.T) {
	account := createRandomAccount(t)

	for i := 0; i < 5; i++ {
		arg := CreateEntryParams{
			AccountID: account.ID,
			Amount:    int64((i + 1) * 1000),
			Currency:  "MAD",
			Type:      "deposit",
			Description: pgtype.Text{
				String: "Test deposit",
				Valid:  true,
			},
		}

		_, err := testQueries.CreateEntry(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	entries, err := testQueries.ListEntriesByAccountID(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Len(t, entries, 5)

	for _, entry := range entries {
		require.Equal(t, account.ID, entry.AccountID)
	}
}



func TestEntriesBelongToCorrectAccount(t *testing.T) {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	arg1 := CreateEntryParams{
		AccountID: account1.ID,
		Amount:    1000,
		Currency:  "MAD",
		Type:      "deposit",
		Description: pgtype.Text{
			String: "Account 1 entry",
			Valid:  true,
		},
	}

	arg2 := CreateEntryParams{
		AccountID: account2.ID,
		Amount:    2000,
		Currency:  "MAD",
		Type:      "deposit",
		Description: pgtype.Text{
			String: "Account 2 entry",
			Valid:  true,
		},
	}

	entry1, err := testQueries.CreateEntry(
		context.Background(),
		arg1,
	)

	require.NoError(t, err)

	entry2, err := testQueries.CreateEntry(
		context.Background(),
		arg2,
	)

	require.NoError(t, err)

	require.Equal(t, account1.ID, entry1.AccountID)
	require.Equal(t, account2.ID, entry2.AccountID)
}

func TestListEntriesByAccountIDWithPagination(t *testing.T) {
	account := createRandomAccount(t)

	for i := 0; i < 10; i++ {
		arg := CreateEntryParams{
			AccountID: account.ID,
			Amount:    int64((i + 1) * 1000),
			Currency:  "MAD",
			Type:      "deposit",
			Description: pgtype.Text{
				String: "Pagination test",
				Valid:  true,
			},
		}

		_, err := testQueries.CreateEntry(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	firstPage, err := testQueries.ListEntriesByAccountIDWithPagination(
		context.Background(),
		ListEntriesByAccountIDWithPaginationParams{
			AccountID: account.ID,
			Limit:     3,
			Offset:    0,
		},
	)

	require.NoError(t, err)
	require.Len(t, firstPage, 3)

	secondPage, err := testQueries.ListEntriesByAccountIDWithPagination(
		context.Background(),
		ListEntriesByAccountIDWithPaginationParams{
			AccountID: account.ID,
			Limit:     3,
			Offset:    3,
		},
	)

	require.NoError(t, err)
	require.Len(t, secondPage, 3)

	require.NotEqual(t, firstPage[0].ID, secondPage[0].ID)
}



func TestEntriesOrderedByCreatedAt(t *testing.T) {
	account := createRandomAccount(t)

	for i := 0; i < 5; i++ {
		arg := CreateEntryParams{
			AccountID: account.ID,
			Amount:    int64((i + 1) * 1000),
			Currency:  "MAD",
			Type:      "deposit",
			Description: pgtype.Text{
				String: "Ordering test",
				Valid:  true,
			},
		}

		_, err := testQueries.CreateEntry(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	entries, err := testQueries.ListEntriesByAccountID(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Len(t, entries, 5)

	for i := 0; i < len(entries)-1; i++ {
		require.True(
			t,
			entries[i].CreatedAt.Time.After(entries[i+1].CreatedAt.Time) ||
				entries[i].CreatedAt.Time.Equal(entries[i+1].CreatedAt.Time),
		)
	}
}

func TestDifferentAccountEntriesDoNotMix(t *testing.T) {
	account1 := createRandomAccount(t)
	account2 := createRandomAccount(t)

	for i := 0; i < 2; i++ {
		arg := CreateEntryParams{
			AccountID: account1.ID,
			Amount:    1000,
			Currency:  "MAD",
			Type:      "deposit",
			Description: pgtype.Text{
				String: "Account 1",
				Valid:  true,
			},
		}

		_, err := testQueries.CreateEntry(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	arg := CreateEntryParams{
		AccountID: account2.ID,
		Amount:    2000,
		Currency:  "MAD",
		Type:      "deposit",
		Description: pgtype.Text{
			String: "Account 2",
			Valid:  true,
		},
	}

	_, err := testQueries.CreateEntry(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	account1Entries, err := testQueries.ListEntriesByAccountID(
		context.Background(),
		account1.ID,
	)

	require.NoError(t, err)
	require.Len(t, account1Entries, 2)

	for _, entry := range account1Entries {
		require.Equal(t, account1.ID, entry.AccountID)
		require.NotEqual(t, account2.ID, entry.AccountID)
	}

	account2Entries, err := testQueries.ListEntriesByAccountID(
		context.Background(),
		account2.ID,
	)

	require.NoError(t, err)
	require.Len(t, account2Entries, 1)
	require.Equal(t, account2.ID, account2Entries[0].AccountID)
}

func TestGetAccountBalance(t *testing.T) {
	account := createRandomAccount(t)

	entries := []CreateEntryParams{
		{
			AccountID: account.ID,
			Amount:    10000,
			Currency:  "MAD",
			Type:      "deposit",
			Description: pgtype.Text{
				String: "Deposit",
				Valid:  true,
			},
		},
		{
			AccountID: account.ID,
			Amount:    3000,
			Currency:  "MAD",
			Type:      "withdrawal",
			Description: pgtype.Text{
				String: "Withdrawal",
				Valid:  true,
			},
		},
		{
			AccountID: account.ID,
			Amount:    2000,
			Currency:  "MAD",
			Type:      "fee",
			Description: pgtype.Text{
				String: "Fee",
				Valid:  true,
			},
		},
	}

	for _, arg := range entries {
		_, err := testQueries.CreateEntry(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	balance, err := testQueries.GetAccountBalanceFromEntries(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Equal(t, int64(5000), balance)
}
