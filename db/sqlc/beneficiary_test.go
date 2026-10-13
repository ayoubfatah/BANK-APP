package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomBeneficiary(t *testing.T, user User) Beneficiary {
	arg := CreateBeneficiaryParams{
		UserID: user.ID,
		Name:   "John Doe",

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

	beneficiary, err := testQueries.CreateBeneficiary(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	require.NotZero(t, beneficiary.ID)
	require.Equal(t, user.ID, beneficiary.UserID)
	require.Equal(t, arg.Name, beneficiary.Name)

	require.Equal(t, arg.AccountNumber, beneficiary.AccountNumber)
	require.Equal(t, arg.Iban, beneficiary.Iban)
	require.Equal(t, arg.Bic, beneficiary.Bic)
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

	require.Equal(t, "123456789", beneficiary.AccountNumber.String)
	require.True(t, beneficiary.AccountNumber.Valid)

	require.Equal(
		t,
		"MA6401151900000123456789012",
		beneficiary.Iban.String,
	)
	require.True(t, beneficiary.Iban.Valid)

	require.Equal(t, "CIHMMAMC", beneficiary.Bic.String)
	require.True(t, beneficiary.Bic.Valid)

	require.Equal(t, "CIH Bank", beneficiary.BankName.String)
	require.True(t, beneficiary.BankName.Valid)

	require.Equal(t, "MA", beneficiary.Country)

	require.Equal(t, "MAD", beneficiary.Currency.String)
	require.True(t, beneficiary.Currency.Valid)
}

func TestGetBeneficiary(t *testing.T) {
	user := createNewUser(t)

	createdBeneficiary := createRandomBeneficiary(t, user)

	arg := GetBeneficiaryParams{
		ID:     createdBeneficiary.ID,
		UserID: user.ID,
	}

	beneficiary, err := testQueries.GetBeneficiary(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	require.Equal(t, createdBeneficiary.ID, beneficiary.ID)
	require.Equal(t, createdBeneficiary.UserID, beneficiary.UserID)
	require.Equal(t, createdBeneficiary.Name, beneficiary.Name)

	require.Equal(
		t,
		createdBeneficiary.AccountNumber,
		beneficiary.AccountNumber,
	)

	require.Equal(
		t,
		createdBeneficiary.Iban,
		beneficiary.Iban,
	)

	require.Equal(
		t,
		createdBeneficiary.Bic,
		beneficiary.Bic,
	)

	require.Equal(
		t,
		createdBeneficiary.BankName,
		beneficiary.BankName,
	)

	require.Equal(
		t,
		createdBeneficiary.Country,
		beneficiary.Country,
	)

	require.Equal(
		t,
		createdBeneficiary.Currency,
		beneficiary.Currency,
	)
}

func TestListBeneficiariesByUserID(t *testing.T) {
	user := createNewUser(t)

	for i := 0; i < 5; i++ {
		arg := CreateBeneficiaryParams{
			UserID: user.ID,
			Name:   "Beneficiary",

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

		_, err := testQueries.CreateBeneficiary(
			context.Background(),
			arg,
		)

		require.NoError(t, err)
	}

	beneficiaries, err := testQueries.ListBeneficiariesByUserID(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, beneficiaries, 5)

	for _, beneficiary := range beneficiaries {
		require.Equal(t, user.ID, beneficiary.UserID)
	}
}

func TestUpdateBeneficiary(t *testing.T) {
	user := createNewUser(t)

	beneficiary := createRandomBeneficiary(t, user)

	arg := UpdateBeneficiaryParams{
		ID:   beneficiary.ID,
		Name: "Jane Doe",

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

		BankName: pgtype.Text{
			String: "Attijariwafa Bank",
			Valid:  true,
		},

		Country: "MA",

		Currency: pgtype.Text{
			String: "EUR",
			Valid:  true,
		},

		UserID: user.ID,
	}

	updatedBeneficiary, err := testQueries.UpdateBeneficiary(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	require.Equal(t, beneficiary.ID, updatedBeneficiary.ID)
	require.Equal(t, user.ID, updatedBeneficiary.UserID)

	require.Equal(t, "Jane Doe", updatedBeneficiary.Name)

	require.Equal(
		t,
		"987654321",
		updatedBeneficiary.AccountNumber.String,
	)

	require.Equal(
		t,
		"MA6401151900000999999999999",
		updatedBeneficiary.Iban.String,
	)

	require.Equal(
		t,
		"ATTIJO123",
		updatedBeneficiary.Bic.String,
	)

	require.Equal(
		t,
		"Attijariwafa Bank",
		updatedBeneficiary.BankName.String,
	)

	require.Equal(t, "MA", updatedBeneficiary.Country)

	require.Equal(
		t,
		"EUR",
		updatedBeneficiary.Currency.String,
	)
}

func TestDeleteBeneficiary(t *testing.T) {
	user := createNewUser(t)

	beneficiary := createRandomBeneficiary(t, user)

	err := testQueries.DeleteBeneficiary(
		context.Background(),
		DeleteBeneficiaryParams{
			ID:     beneficiary.ID,
			UserID: user.ID,
		},
	)

	require.NoError(t, err)

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
	arg := CreateBeneficiaryParams{
		UserID: 999999999,
		Name:   "John Doe",

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

	beneficiary, err := testQueries.CreateBeneficiary(
		context.Background(),
		arg,
	)

	require.Error(t, err)
	require.Empty(t, beneficiary)
}
