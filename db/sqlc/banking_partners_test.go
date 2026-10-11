package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateBankingPartner(t *testing.T) {
	arg := CreateBankingPartnerParams{
		CurrencyCode: "EUR",
		CountryCode:  "FR",
		ProviderName: "Test Create Provider",
		DetailType:   "IBAN",
	}

	partner, err := testQueries.CreateBankingPartner(context.Background(), arg)
	require.NoError(t, err)
	require.NotEmpty(t, partner)

	require.Equal(t, arg.CurrencyCode, partner.CurrencyCode)
	require.Equal(t, arg.CountryCode, partner.CountryCode)
	require.Equal(t, arg.ProviderName, partner.ProviderName)
	require.Equal(t, arg.DetailType, partner.DetailType)
	require.True(t, partner.IsActive)
}

func TestGetBankingPartnersByCurrencyAndCountry(t *testing.T) {
	ctx := context.Background()

	// EUR/FR is already present in the seed.
	partners, err := testQueries.GetBankingPartnersByCurrencyAndCountry(
		ctx,
		GetBankingPartnersByCurrencyAndCountryParams{
			CurrencyCode: "EUR",
			CountryCode:  "FR",
		},
	)

	require.NoError(t, err)
	require.NotEmpty(t, partners)

	found := false
	for _, partner := range partners {
		if partner.ProviderName == "Demo SEPA Provider" &&
			partner.DetailType == "IBAN" {
			found = true
			require.Equal(t, "EUR", partner.CurrencyCode)
			require.Equal(t, "FR", partner.CountryCode)
			require.True(t, partner.IsActive)
			break
		}
	}

	require.True(t, found, "expected seeded EUR/FR provider")
}

func TestListBankingPartners(t *testing.T) {
	partners, err := testQueries.ListBankingPartners(context.Background())

	require.NoError(t, err)
	require.NotEmpty(t, partners)

	for i := 1; i < len(partners); i++ {
		previous := partners[i-1]
		current := partners[i]

		ordered := previous.CurrencyCode < current.CurrencyCode ||
			(previous.CurrencyCode == current.CurrencyCode &&
				previous.CountryCode < current.CountryCode) ||
			(previous.CurrencyCode == current.CurrencyCode &&
				previous.CountryCode == current.CountryCode &&
				previous.ProviderName <= current.ProviderName)

		require.True(t, ordered, "banking partners should be ordered")
	}
}