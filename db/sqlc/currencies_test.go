package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// test USD
func TestCreateUSD(t *testing.T) {
	currency, err := testQueries.GetCurrency(context.Background(), "USD")
	require.NoError(t, err)
	require.Equal(t, "USD", currency.Code)
	require.Equal(t, "US Dollar", currency.Name)
	require.Equal(t, "$", currency.Symbol)
	require.Equal(t, int32(2), currency.Decimals)
	require.True(t, currency.IsActive)
}

// CREATE EUR
func TestCreateEUR(t *testing.T) {

	currency, err := testQueries.GetCurrency(context.Background(), "EUR")
	require.NoError(t, err)
	require.Equal(t, "EUR", currency.Code)
	require.Equal(t, "Euro", currency.Name)
	require.Equal(t, "€", currency.Symbol)
	require.Equal(t, int32(2), currency.Decimals)
	require.True(t, currency.IsActive)
}

// GET CURRENCY BY CODE
func TestGetCurrency(t *testing.T) {
	currency, err := testQueries.GetCurrency(context.Background(), "GBP")
	require.NoError(t, err)
	fetched, err := testQueries.GetCurrency(
		context.Background(),
		currency.Code,
	)

	require.NoError(t, err)
	require.NotEmpty(t, fetched)

	require.Equal(t, currency.Code, fetched.Code)
	require.Equal(t, currency.Name, fetched.Name)
	require.Equal(t, currency.Symbol, fetched.Symbol)
	require.Equal(t, currency.Decimals, fetched.Decimals)
	require.Equal(t, currency.IsActive, fetched.IsActive)
}

// INVALID CURRENCY RETURNS NOT FOUND
func TestGetInvalidCurrency(t *testing.T) {
	_, err := testQueries.GetCurrency(
		context.Background(),
		"XYZ",
	)

	require.Error(t, err)
}

// UPDATE CURRENCY
func TestUpdateCurrency(t *testing.T) {

	_, err := testQueries.GetCurrency(context.Background(), "GBP")
	require.NoError(t, err)
	updated, err := testQueries.UpdateCurrency(
		context.Background(),
		UpdateCurrencyParams{
			Code:     "CAD",
			Name:     "Canadian Dollar Updated",
			Symbol:   "CA$",
			Decimals: 2,
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, updated)

	require.Equal(t, "CAD", updated.Code)
	require.Equal(t, "Canadian Dollar Updated", updated.Name)
	require.Equal(t, "CA$", updated.Symbol)
	require.Equal(t, int32(2), updated.Decimals)
	require.True(t, updated.IsActive)
}

// DEACTIVATE CURRENCY
func TestDeactivateCurrency(t *testing.T) {
	currency, err := testQueries.GetCurrency(context.Background(), "AUD")
	updated, err := testQueries.UpdateCurrencyActiveState(
		context.Background(),
		UpdateCurrencyActiveStateParams{
			Code:     currency.Code,
			IsActive: true,
		},
	)

	require.NoError(t, err)

	require.True(t, updated.IsActive)

	deactivated, err := testQueries.UpdateCurrencyActiveState(
		context.Background(),
		UpdateCurrencyActiveStateParams{
			Code:     currency.Code,
			IsActive: false,
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, deactivated)

	require.Equal(t, currency.Code, deactivated.Code)
	require.False(t, deactivated.IsActive)
}
