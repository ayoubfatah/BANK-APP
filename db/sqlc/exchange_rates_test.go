package db

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestGetExchangeRate(t *testing.T) {
	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "USD",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.NotZero(t, rate.ID)
	require.Equal(t, "USD", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)

	require.True(t, rate.Rate.Valid)
	require.True(t, rate.Source.Valid)

	require.Equal(t, "manual_seed", rate.Source.String)
}

func TestGetLatestExchangeRate(t *testing.T) {
	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "USD",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.NotZero(t, rate.ID)
	require.Equal(t, "USD", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)
	require.Equal(t, "manual_seed", rate.Source.String)
	require.True(t, rate.Rate.Valid)
}

func TestGetExchangeRateDifferentCurrencyPair(t *testing.T) {
	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "GBP",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.NotZero(t, rate.ID)
	require.Equal(t, "GBP", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)
	require.Equal(t, "manual_seed", rate.Source.String)
	require.True(t, rate.Rate.Valid)
}

func TestGetExchangeRateInvalidCurrency(t *testing.T) {
	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "XXX",
			QuoteCurrency: "EUR",
		},
	)

	require.Error(t, err)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	require.Empty(t, rate)
}

func TestListExchangeRates(t *testing.T) {
	rates, err := testQueries.ListExchangeRates(
		context.Background(),
		ListExchangeRatesParams{
			BaseCurrency:  "GBP",
			QuoteCurrency: "JPY",
			Limit:         10,
			Offset:        0,
		},
	)

	require.NoError(t, err)

	// Seed contains exactly one GBP -> JPY rate.
	require.Len(t, rates, 1)

	rate := rates[0]

	require.NotZero(t, rate.ID)
	require.Equal(t, "GBP", rate.BaseCurrency)
	require.Equal(t, "JPY", rate.QuoteCurrency)
	require.Equal(t, "manual_seed", rate.Source.String)
	require.True(t, rate.Rate.Valid)
	require.True(t, rate.Source.Valid)
}
