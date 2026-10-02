package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createExRate(
	t *testing.T,
	baseCurrency string,
	quoteCurrency string,
	rate string,
	source string,
	fetchedAt time.Time,
) ExchangeRate {
	arg := CreateExchangeRateParams{
		BaseCurrency:  baseCurrency,
		QuoteCurrency: quoteCurrency,
		Rate: pgtype.Numeric{
			Int:   nil,
			Exp:   0,
			NaN:   false,
			Valid: true,
		},
		Source: pgtype.Text{
			String: source,
			Valid:  true,
		},
	
	}

	// Parse the decimal string into pgtype.Numeric.
	err := arg.Rate.Scan(rate)
	require.NoError(t, err)

	exchangeRate, err := testQueries.CreateExchangeRate(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotZero(t, exchangeRate.ID)

	require.Equal(t, baseCurrency, exchangeRate.BaseCurrency)
	require.Equal(t, quoteCurrency, exchangeRate.QuoteCurrency)

	require.True(t, exchangeRate.Rate.Valid)
	require.True(t, exchangeRate.Source.Valid)
	require.True(t, exchangeRate.FetchedAt.Valid)

	return exchangeRate
}

func TestCreateExchangeRate(t *testing.T) {
	fetchedAt := time.Now().UTC()

	exchangeRate := createExRate(
		t,
		"USD",
		"EUR",
		"0.9200000000",
		"test-provider",
		fetchedAt,
	)

	require.Equal(t, "USD", exchangeRate.BaseCurrency)
	require.Equal(t, "EUR", exchangeRate.QuoteCurrency)

	require.True(t, exchangeRate.Rate.Valid)
	require.True(t, exchangeRate.Source.Valid)
	require.True(t, exchangeRate.FetchedAt.Valid)

	require.Equal(t, "test-provider", exchangeRate.Source.String)
	require.WithinDuration(
		t,
		fetchedAt,
		exchangeRate.FetchedAt.Time,
		time.Microsecond,
	)
}

func TestGetExchangeRate(t *testing.T) {
	fetchedAt := time.Now().UTC().Truncate(time.Microsecond)

	createdRate := createExRate(
		t,
		"USD",
		"EUR",
		"0.9200000000",
		"test-provider",
		fetchedAt,
	)

	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "USD",
			QuoteCurrency: "EUR",
			
		},
	)

	require.NoError(t, err)

	require.Equal(t, createdRate.ID, rate.ID)
	require.Equal(t, "USD", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)
	require.Equal(t, "test-provider", rate.Source.String)

	require.True(t, rate.Rate.Valid)
	require.True(t, rate.FetchedAt.Valid)
}

func TestGetLatestExchangeRate(t *testing.T) {
	now := time.Now().UTC()

	olderRate := createExRate(
		t,
		"USD",
		"EUR",
		"0.9100000000",
		"test-provider",
		now.Add(-2*time.Hour),
	)

	latestRate := createExRate(
		t,
		"USD",
		"EUR",
		"0.9200000000",
		"test-provider",
		now,
	)

	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "USD",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.Equal(t, latestRate.ID, rate.ID)
	require.NotEqual(t, olderRate.ID, rate.ID)

	require.Equal(t, "USD", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)
	require.Equal(t, "test-provider", rate.Source.String)

	require.True(t, rate.Rate.Valid)
	require.True(t, rate.FetchedAt.Valid)

	require.Equal(
		t,
		latestRate.FetchedAt.Time,
		rate.FetchedAt.Time,
	)
}

func TestGetLatestExchangeRateDifferentCurrencyPair(t *testing.T) {
	now := time.Now().UTC()

	createExRate(
		t,
		"USD",
		"EUR",
		"0.9200000000",
		"test-provider",
		now,
	)

	GBPToEUR := createExRate(
		t,
		"GBP",
		"EUR",
		"1.1700000000",
		"test-provider",
		now,
	)

	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "GBP",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.Equal(t, GBPToEUR.ID, rate.ID)
	require.Equal(t, "GBP", rate.BaseCurrency)
	require.Equal(t, "EUR", rate.QuoteCurrency)
}

func TestGetLatestExchangeRateMultipleRates(t *testing.T) {
	now := time.Now().UTC()

	createExRate(
		t,
		"USD",
		"EUR",
		"0.9000000000",
		"test-provider",
		now.Add(-4*time.Hour),
	)

	createExRate(
		t,
		"USD",
		"EUR",
		"0.9100000000",
		"test-provider",
		now.Add(-3*time.Hour),
	)

	createExRate(
		t,
		"USD",
		"EUR",
		"0.9200000000",
		"test-provider",
		now.Add(-2*time.Hour),
	)

	latestRate := createExRate(
		t,
		"USD",
		"EUR",
		"0.9300000000",
		"test-provider",
		now,
	)

	rate, err := testQueries.GetExchangeRate(
		context.Background(),
		GetExchangeRateParams{
			BaseCurrency:  "USD",
			QuoteCurrency: "EUR",
		},
	)

	require.NoError(t, err)

	require.Equal(t, latestRate.ID, rate.ID)
	require.Equal(t, latestRate.FetchedAt.Time, rate.FetchedAt.Time)
}

func TestGetLatestExchangeRateInvalidCurrency(t *testing.T) {
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
	now := time.Now().UTC()

	createExRate(
		t,
		"GBP",
		"JPY",
		"190.0000000000",
		"test-provider",
		now.Add(-3*time.Hour),
	)

	createExRate(
		t,
		"GBP",
		"JPY",
		"191.0000000000",
		"test-provider",
		now.Add(-2*time.Hour),
	)

	createExRate(
		t,
		"GBP",
		"JPY",
		"192.0000000000",
		"test-provider",
		now.Add(-1*time.Hour),
	)

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
	require.Len(t, rates, 3)

	// Newest first.
	require.Equal(t, "GBP", rates[0].BaseCurrency)
	require.Equal(t, "JPY", rates[0].QuoteCurrency)

	require.Equal(t, "GBP", rates[1].BaseCurrency)
	require.Equal(t, "JPY", rates[1].QuoteCurrency)

	require.Equal(t, "GBP", rates[2].BaseCurrency)
	require.Equal(t, "JPY", rates[2].QuoteCurrency)

	require.True(
		t,
		rates[0].FetchedAt.Time.After(rates[1].FetchedAt.Time),
	)

	require.True(
		t,
		rates[1].FetchedAt.Time.After(rates[2].FetchedAt.Time),
	)
}