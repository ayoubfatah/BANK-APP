package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestCreateCardSpendingLimits(t *testing.T) {
	card := createNewCard(t)

	limits, err := testQueries.CreateCardSpendingLimits(
		context.Background(),
		CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 500000,
				Valid: true,
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, card.ID, limits.CardID)
	require.True(t, limits.DailyLimit.Valid)
	require.Equal(t, int64(50000), limits.DailyLimit.Int64)
	require.True(t, limits.MonthlyLimit.Valid)
	require.Equal(t, int64(500000), limits.MonthlyLimit.Int64)
	require.False(t, limits.CreatedAt.Time.IsZero())
	require.False(t, limits.UpdatedAt.Time.IsZero())
}

func TestGetCardSpendingLimits(t *testing.T) {
	card := createNewCard(t)

	createdLimits, err := testQueries.CreateCardSpendingLimits(
		context.Background(),
		CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 500000,
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	limits, err := testQueries.GetCardSpendingLimits(
		context.Background(),
		GetCardSpendingLimitsParams{
			CardID: card.ID,
			UserID: card.UserID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, createdLimits.CardID, limits.CardID)
	require.Equal(t, createdLimits.DailyLimit, limits.DailyLimit)
	require.Equal(t, createdLimits.MonthlyLimit, limits.MonthlyLimit)
}

func TestGetCardSpendingLimitsWrongUser(t *testing.T) {
	card := createNewCard(t)
	anotherUser := createNewUser(t)

	_, err := testQueries.CreateCardSpendingLimits(
		context.Background(),
		CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 500000,
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	_, err = testQueries.GetCardSpendingLimits(
		context.Background(),
		GetCardSpendingLimitsParams{
			CardID: card.ID,
			UserID: anotherUser.ID,
		},
	)

	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestUpdateCardSpendingLimits(t *testing.T) {
	card := createNewCard(t)

	_, err := testQueries.CreateCardSpendingLimits(
		context.Background(),
		CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 500000,
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	updatedLimits, err := testQueries.UpdateCardSpendingLimits(
		context.Background(),
		UpdateCardSpendingLimitsParams{
			CardID: card.ID,
			UserID: card.UserID,
			DailyLimit: pgtype.Int8{
				Int64: 75000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 750000,
				Valid: true,
			},
		},
	)

	require.NoError(t, err)
	require.Equal(t, int64(75000), updatedLimits.DailyLimit.Int64)
	require.Equal(t, int64(750000), updatedLimits.MonthlyLimit.Int64)
	require.True(t, updatedLimits.DailyLimit.Valid)
	require.True(t, updatedLimits.MonthlyLimit.Valid)
	require.True(t,
		updatedLimits.UpdatedAt.Time.After(
			updatedLimits.CreatedAt.Time,
		) || updatedLimits.UpdatedAt.Time.Equal(
			updatedLimits.CreatedAt.Time,
		),
	)
}

func TestUpdateCardSpendingLimitsWrongUser(t *testing.T) {
	card := createNewCard(t)
	anotherUser := createNewUser(t)

	originalLimits, err := testQueries.CreateCardSpendingLimits(
		context.Background(),
		CreateCardSpendingLimitsParams{
			CardID: card.ID,
			DailyLimit: pgtype.Int8{
				Int64: 50000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 500000,
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	_, err = testQueries.UpdateCardSpendingLimits(
		context.Background(),
		UpdateCardSpendingLimitsParams{
			CardID: card.ID,
			UserID: anotherUser.ID,
			DailyLimit: pgtype.Int8{
				Int64: 100000,
				Valid: true,
			},
			MonthlyLimit: pgtype.Int8{
				Int64: 1000000,
				Valid: true,
			},
		},
	)

	require.ErrorIs(t, err, pgx.ErrNoRows)

	limits, err := testQueries.GetCardSpendingLimits(
		context.Background(),
		GetCardSpendingLimitsParams{
			CardID: card.ID,
			UserID: card.UserID,
		},
	)

	require.NoError(t, err)
	require.Equal(t, originalLimits.DailyLimit, limits.DailyLimit)
	require.Equal(t, originalLimits.MonthlyLimit, limits.MonthlyLimit)
}
