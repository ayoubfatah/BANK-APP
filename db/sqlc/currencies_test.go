package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func createCurr(t *testing.T, code, name, symbol string) Currency {
	err := testQueries.DeleteAllCurrencies(context.Background())
	require.NoError(t, err)
	arg := CreateCurrencyParams{
		Code:     code,
		Name:     name,
		Symbol:   symbol,
		Decimals: 2,
		IsActive: true,
	}

	currency, err := testQueries.CreateCurrency(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotEmpty(t, currency)

	return currency
}

// CREATE USD
func TestCreateUSD(t *testing.T) {
	currency := createCurr(t, "USD", "US Dollar", "$")

	require.Equal(t, "USD", currency.Code)
	require.Equal(t, "US Dollar", currency.Name)
	require.Equal(t, "$", currency.Symbol)
	require.Equal(t, int32(2), currency.Decimals)
	require.True(t, currency.IsActive)
}

// CREATE EUR
func TestCreateEUR(t *testing.T) {
	currency := createCurr(t, "EUR", "Euro", "€")

	require.Equal(t, "EUR", currency.Code)
	require.Equal(t, "Euro", currency.Name)
	require.Equal(t, "€", currency.Symbol)
	require.Equal(t, int32(2), currency.Decimals)
	require.True(t, currency.IsActive)
}

// GET CURRENCY BY CODE
func TestGetCurrency(t *testing.T) {
	currency := createCurr(t, "GBP", "British Pound", "£")

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

// LIST CURRENCIES
func TestListCurrencies(t *testing.T) {
	createCurr(t, "MAD", "Moroccan Dirham", "DH")
	createCurr(t, "JPY", "Japanese Yen", "¥")

	currencies, err := testQueries.ListCurrencies(
		context.Background(),
		ListCurrenciesParams{
			Limit:  10,
			Offset: 0,
		},
	)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(currencies), 2)

	for _, currency := range currencies {
		require.NotEmpty(t, currency.Code)
		require.NotEmpty(t, currency.Name)
		require.NotEmpty(t, currency.Symbol)
		require.True(t, currency.IsActive)
	}
}

// UPDATE CURRENCY
func TestUpdateCurrency(t *testing.T) {
	createCurr(t, "CAD", "Canadian Dollar", "C$")

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

// DUPLICATE CURRENCY CODE FAILS
func TestDuplicateCurrencyCode(t *testing.T) {
	createCurr(t, "CHF", "Swiss Franc", "CHF")

	arg := CreateCurrencyParams{
		Code:     "CHF",
		Name:     "Another Swiss Franc",
		Symbol:   "CHF",
		Decimals: 2,
		IsActive: true,
	}

	_, err := testQueries.CreateCurrency(
		context.Background(),
		arg,
	)

	require.Error(t, err)
}

// DEACTIVATE CURRENCY
func TestDeactivateCurrency(t *testing.T) {
	currency := createCurr(t, "AUD", "Australian Dollar", "A$")

	require.True(t, currency.IsActive)

	deactivated, err := testQueries.DeactivateCurrency(
		context.Background(),
		currency.Code,
	)

	require.NoError(t, err)
	require.NotEmpty(t, deactivated)

	require.Equal(t, currency.Code, deactivated.Code)
	require.False(t, deactivated.IsActive)
}