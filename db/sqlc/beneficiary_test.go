package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomBeneficiary(t *testing.T, user User) Beneficiary {
	t.Helper()

	arg := CreateBeneficiaryParams{
		UserID: user.ID,
		Name:   "John Doe",
		Nickname: pgtype.Text{
			String: "John",
			Valid:  true,
		},
		AccountNumber: pgtype.Text{
			String: "123456789",
			Valid:  true,
		},
		Iban: pgtype.Text{
			String: "MA6401151900000123456789012",
			Valid:  true,
		},
		Bic: pgtype.Text{
			String: "CIHMMAMC",
			Valid:  true,
		},
		RoutingNumber: pgtype.Text{
			String: "123456789",
			Valid:  true,
		},
		BankName: pgtype.Text{
			String: "CIH Bank",
			Valid:  true,
		},
		Country: "MA",
		Currency: pgtype.Text{
			String: "MAD",
			Valid:  true,
		},
	}

	beneficiary, err := testQueries.CreateBeneficiary(context.Background(), arg)
	require.NoError(t, err)

	require.NotZero(t, beneficiary.ID)
	require.Equal(t, user.ID, beneficiary.UserID)
	require.Equal(t, arg.Name, beneficiary.Name)
	require.Equal(t, arg.Nickname, beneficiary.Nickname)
	require.Equal(t, arg.AccountNumber, beneficiary.AccountNumber)
	require.Equal(t, arg.Iban, beneficiary.Iban)
	require.Equal(t, arg.Bic, beneficiary.Bic)
	require.Equal(t, arg.RoutingNumber, beneficiary.RoutingNumber)
	require.Equal(t, arg.BankName, beneficiary.BankName)
	require.Equal(t, arg.Country, beneficiary.Country)
	require.Equal(t, arg.Currency, beneficiary.Currency)
	require.NotZero(t, beneficiary.CreatedAt)
	require.NotZero(t, beneficiary.UpdatedAt)

	return beneficiary
}

func TestCreateBeneficiary(t *testing.T) {
	user := createNewUser(t)
	beneficiary := createRandomBeneficiary(t, user)

	require.Equal(t, user.ID, beneficiary.UserID)
	require.Equal(t, "John Doe", beneficiary.Name)
	require.Equal(t, "John", beneficiary.Nickname.String)
	require.True(t, beneficiary.Nickname.Valid)

	require.Equal(t, "123456789", beneficiary.AccountNumber.String)
	require.True(t, beneficiary.AccountNumber.Valid)

	require.Equal(t, "MA6401151900000123456789012", beneficiary.Iban.String)
	require.True(t, beneficiary.Iban.Valid)

	require.Equal(t, "CIHMMAMC", beneficiary.Bic.String)
	require.True(t, beneficiary.Bic.Valid)

	require.Equal(t, "123456789", beneficiary.RoutingNumber.String)
	require.True(t, beneficiary.RoutingNumber.Valid)

	require.Equal(t, "CIH Bank", beneficiary.BankName.String)
	require.True(t, beneficiary.BankName.Valid)

	require.Equal(t, "MA", beneficiary.Country)
	require.Equal(t, "MAD", beneficiary.Currency.String)
	require.True(t, beneficiary.Currency.Valid)
}

func TestGetBeneficiary(t *testing.T) {
	user := createNewUser(t)
	created := createRandomBeneficiary(t, user)

	beneficiary, err := testQueries.GetBeneficiary(
		context.Background(),
		GetBeneficiaryParams{
			ID:     created.ID,
			UserID: user.ID,
		},
	)
	require.NoError(t, err)

	require.Equal(t, created.ID, beneficiary.ID)
	require.Equal(t, created.UserID, beneficiary.UserID)
	require.Equal(t, created.Name, beneficiary.Name)
	require.Equal(t, created.Nickname, beneficiary.Nickname)
	require.Equal(t, created.AccountNumber, beneficiary.AccountNumber)
	require.Equal(t, created.Iban, beneficiary.Iban)
	require.Equal(t, created.Bic, beneficiary.Bic)
	require.Equal(t, created.RoutingNumber, beneficiary.RoutingNumber)
	require.Equal(t, created.BankName, beneficiary.BankName)
	require.Equal(t, created.Country, beneficiary.Country)
	require.Equal(t, created.Currency, beneficiary.Currency)
}

func TestListBeneficiariesByUserID(t *testing.T) {
	user := createNewUser(t)

	for i := 0; i < 5; i++ {
		_, err := testQueries.CreateBeneficiary(
			context.Background(),
			CreateBeneficiaryParams{
				UserID: user.ID,
				Name:   "Beneficiary",
				Nickname: pgtype.Text{
					String: "Contact",
					Valid:  true,
				},
				AccountNumber: pgtype.Text{
					String: "ACCOUNT-" + string(rune('A'+i)),
					Valid:  true,
				},
				Iban: pgtype.Text{
					String: "IBAN-" + string(rune('A'+i)),
					Valid:  true,
				},
				Bic: pgtype.Text{
					String: "BIC-" + string(rune('A'+i)),
					Valid:  true,
				},
				RoutingNumber: pgtype.Text{
					String: "ROUTING-" + string(rune('A'+i)),
					Valid:  true,
				},
				BankName: pgtype.Text{
					String: "CIH Bank",
					Valid:  true,
				},
				Country: "MA",
				Currency: pgtype.Text{
					String: "MAD",
					Valid:  true,
				},
			},
		)
		require.NoError(t, err)
	}

	beneficiaries, err := testQueries.ListBeneficiariesByUserID(
		context.Background(),
		ListBeneficiariesByUserIDParams{
			UserID: user.ID,
			Search: pgtype.Text{},
		},
	)
	require.NoError(t, err)
	require.Len(t, beneficiaries, 5)

	for _, beneficiary := range beneficiaries {
		require.Equal(t, user.ID, beneficiary.UserID)
	}
}

func TestListBeneficiariesByUserIDWithSearch(t *testing.T) {
	user := createNewUser(t)

	createRandomBeneficiary(t, user)

	_, err := testQueries.CreateBeneficiary(
		context.Background(),
		CreateBeneficiaryParams{
			UserID: user.ID,
			Name:   "Jane Smith",
			Nickname: pgtype.Text{
				String: "Jane",
				Valid:  true,
			},
			BankName: pgtype.Text{
				String: "Attijariwafa Bank",
				Valid:  true,
			},
			Country: "MA",
			Currency: pgtype.Text{
				String: "MAD",
				Valid:  true,
			},
		},
	)
	require.NoError(t, err)

	beneficiaries, err := testQueries.ListBeneficiariesByUserID(
		context.Background(),
		ListBeneficiariesByUserIDParams{
			UserID: user.ID,
			Search: pgtype.Text{
				String: "Jane",
				Valid:  true,
			},
		},
	)
	require.NoError(t, err)
	require.Len(t, beneficiaries, 1)
	require.Equal(t, "Jane Smith", beneficiaries[0].Name)
}

func TestUpdateBeneficiary(t *testing.T) {
	user := createNewUser(t)
	created := createRandomBeneficiary(t, user)

	updated, err := testQueries.UpdateBeneficiary(
		context.Background(),
		UpdateBeneficiaryParams{
			ID: created.ID,
			Name: pgtype.Text{
				String: "Jane Doe",
				Valid:  true,
			},
			Nickname: pgtype.Text{
				String: "Jane",
				Valid:  true,
			},
			AccountNumber: pgtype.Text{
				String: "987654321",
				Valid:  true,
			},
			Iban: pgtype.Text{
				String: "MA6401151900000999999999999",
				Valid:  true,
			},
			Bic: pgtype.Text{
				String: "ATTIJO123",
				Valid:  true,
			},
			RoutingNumber: pgtype.Text{
				String: "987654321",
				Valid:  true,
			},
			BankName: pgtype.Text{
				String: "Attijariwafa Bank",
				Valid:  true,
			},
			Country: pgtype.Text{
				String: "MA",
				Valid:  true,
			},
			Currency: pgtype.Text{
				String: "EUR",
				Valid:  true,
			},
			UserID: user.ID,
		},
	)
	require.NoError(t, err)

	require.Equal(t, created.ID, updated.ID)
	require.Equal(t, user.ID, updated.UserID)
	require.Equal(t, "Jane Doe", updated.Name)
	require.Equal(t, "Jane", updated.Nickname.String)
	require.Equal(t, "987654321", updated.AccountNumber.String)
	require.Equal(t, "MA6401151900000999999999999", updated.Iban.String)
	require.Equal(t, "ATTIJO123", updated.Bic.String)
	require.Equal(t, "987654321", updated.RoutingNumber.String)
	require.Equal(t, "Attijariwafa Bank", updated.BankName.String)
	require.Equal(t, "MA", updated.Country)
	require.Equal(t, "EUR", updated.Currency.String)
}

func TestDeleteBeneficiary(t *testing.T) {
	user := createNewUser(t)
	beneficiary := createRandomBeneficiary(t, user)

	rowsAffected, err := testQueries.DeleteBeneficiary(
		context.Background(),
		DeleteBeneficiaryParams{
			ID:     beneficiary.ID,
			UserID: user.ID,
		},
	)
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	_, err = testQueries.GetBeneficiary(
		context.Background(),
		GetBeneficiaryParams{
			ID:     beneficiary.ID,
			UserID: user.ID,
		},
	)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestUserCannotAccessAnotherUsersBeneficiary(t *testing.T) {
	user1 := createNewUser(t)
	user2 := createNewUser(t)

	beneficiary := createRandomBeneficiary(t, user1)

	_, err := testQueries.GetBeneficiary(
		context.Background(),
		GetBeneficiaryParams{
			ID:     beneficiary.ID,
			UserID: user2.ID,
		},
	)
	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestCreateBeneficiaryInvalidUser(t *testing.T) {
	beneficiary, err := testQueries.CreateBeneficiary(
		context.Background(),
		CreateBeneficiaryParams{
			UserID:  999999999,
			Name:    "John Doe",
			Country: "MA",
			Currency: pgtype.Text{
				String: "MAD",
				Valid:  true,
			},
		},
	)

	require.Error(t, err)
	require.Empty(t, beneficiary)
}
