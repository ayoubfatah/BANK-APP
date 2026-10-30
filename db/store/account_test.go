package store

import (
	"context"
	"strings"
	"testing"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/stretchr/testify/require"
)

// ---------- helpers ----------

func getActivePartner(t *testing.T, store Store, currency, country string) db.BankingPartner {
	partners, err := store.GetBankingPartnersByCurrencyAndCountry(
		context.Background(),
		db.GetBankingPartnersByCurrencyAndCountryParams{
			CurrencyCode: currency,
			CountryCode:  country,
		},
	)
	require.NoError(t, err)
	require.NotEmpty(t, partners, "no seeded partner for %s/%s", currency, country)

	for _, p := range partners {
		if p.IsActive && p.DetailType != "ROUTING_NUMBER" {
			return p
		}
	}
	t.Fatalf("no usable active partner for %s/%s", currency, country)
	return db.BankingPartner{}
}

func countAccounts(t *testing.T, store Store, userID int64) int {
	var n int
	err := testDB.QueryRow(context.Background(),
		`SELECT count(*) FROM accounts WHERE user_id = $1`, userID).Scan(&n)
	require.NoError(t, err)
	return n
}

func countAccountDetails(t *testing.T, store Store, accountID int64) int {
	var n int
	err := testDB.QueryRow(context.Background(),
		`SELECT count(*) FROM account_details WHERE account_id = $1`, accountID).Scan(&n)
	require.NoError(t, err)
	return n
}

// ---------- tests ----------

// Runs the happy path against every seeded currency/country.
func TestCreateAccountTx(t *testing.T) {
	cases := []struct {
		currency, country string
	}{
		{"EUR", "FR"},
		{"EUR", "DE"},
		{"EUR", "ES"},
		{"EUR", "NL"},
		{"GBP", "GB"},
		{"USD", "US"},
		{"MAD", "MA"},
		{"CAD", "CA"},
		{"JPY", "JP"},
	}

	for _, tc := range cases {

		t.Run(tc.currency+"_"+tc.country, func(t *testing.T) {
			ctx := context.Background()
			store := testStore

			user := createNewUser(t) // fresh user, so no duplicate-currency clash
			partner := getActivePartner(t, store, tc.currency, tc.country)
			_, err := testDB.Exec(
				context.Background(),
				`INSERT INTO kyc_verifications (
        user_id,
        document_type,
        document_number,
        status,
        rejection_reason,
        verified_at
    )
    VALUES ($1, $2, $3, $4, $5, $6)`,
				user.ID,
				"passport",
				"12123131",
				"approved",
				"",
				time.Now().UTC(),
			)

			result, err := store.CreateAccountTx(ctx, CreateAccountTxArguments{
				UserID:           user.ID,
				CurrencyCode:     tc.currency,
				BankingPartnerID: partner.ID,
				Balance:          0,
			})
			require.NoError(t, err)

			// account
			require.NotZero(t, result.Account.ID)
			require.Equal(t, user.ID, result.Account.UserID)
			require.Equal(t, tc.currency, result.Account.Currency)
			require.Equal(t, int64(0), result.Account.Balance)
			require.Equal(t, "active", result.Account.Status)

			// details
			require.NotZero(t, result.AccountDetails.ID)
			require.Equal(t, result.Account.ID, result.AccountDetails.AccountID)
			require.Equal(t, partner.DetailType, result.AccountDetails.Type)
			require.NotEmpty(t, result.AccountDetails.Value)

			if partner.DetailType == "IBAN" {
				require.True(t, strings.HasPrefix(result.AccountDetails.Value, tc.country),
					"IBAN %s should start with %s", result.AccountDetails.Value, tc.country)
			}

			// partner
			require.Equal(t, partner.ID, result.BankPartner.ID)

			// really persisted
			require.Equal(t, 1, countAccounts(t, store, user.ID))
			require.Equal(t, 1, countAccountDetails(t, store, result.Account.ID))
		})
	}
}

func TestCreateAccountTx_InvalidCurrency(t *testing.T) {
	store := testStore
	user := createNewUser(t)
	partner := getActivePartner(t, store, "EUR", "DE")

	_, err := testDB.Exec(
		context.Background(),
		`INSERT INTO kyc_verifications (
        user_id,
        document_type,
        document_number,
        status,
        rejection_reason,
        verified_at
    )
    VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID,
		"passport",
		"12123131",
		"approved",
		"",
		time.Now().UTC(),
	)

	_, err = store.CreateAccountTx(context.Background(), CreateAccountTxArguments{
		UserID:           user.ID,
		CurrencyCode:     "ZZZ",
		BankingPartnerID: partner.ID,
	})
	require.Error(t, err)
	require.Equal(t, 0, countAccounts(t, store, user.ID))
}

func TestCreateAccountTx_PartnerNotFound(t *testing.T) {
	store := testStore
	user := createNewUser(t)

	_, err := testDB.Exec(
		context.Background(),
		`INSERT INTO kyc_verifications (
        user_id,
        document_type,
        document_number,
        status,
        rejection_reason,
        verified_at
    )
    VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID,
		"passport",
		"12123131",
		"approved",
		"",
		time.Now().UTC(),
	)
	_, err = store.CreateAccountTx(context.Background(), CreateAccountTxArguments{
		UserID:           user.ID,
		CurrencyCode:     "EUR",
		BankingPartnerID: 999999999,
	})
	require.Error(t, err)
	require.Equal(t, 0, countAccounts(t, store, user.ID))
}

func TestCreateAccountTx_DuplicateCurrency(t *testing.T) {
	store := testStore
	user := createNewUser(t)
	de := getActivePartner(t, store, "EUR", "DE")
	fr := getActivePartner(t, store, "EUR", "FR")

	_, err := testDB.Exec(
		context.Background(),
		`INSERT INTO kyc_verifications (
        user_id,
        document_type,
        document_number,
        status,
        rejection_reason,
        verified_at
    )
    VALUES ($1, $2, $3, $4, $5, $6)`,
		user.ID,
		"passport",
		"12123131",
		"approved",
		"",
		time.Now().UTC(),
	)
	first, err := store.CreateAccountTx(context.Background(), CreateAccountTxArguments{
		UserID: user.ID, CurrencyCode: "EUR", BankingPartnerID: de.ID,
	})
	require.NoError(t, err)

	// UNIQUE (user_id, currency) must reject a second EUR account,
	// even through a different partner
	_, err = store.CreateAccountTx(context.Background(), CreateAccountTxArguments{
		UserID: user.ID, CurrencyCode: "EUR", BankingPartnerID: fr.ID,
	})
	require.Error(t, err)

	require.Equal(t, 1, countAccounts(t, store, user.ID))
	require.Equal(t, 1, countAccountDetails(t, store, first.Account.ID))
}
