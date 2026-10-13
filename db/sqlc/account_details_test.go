package db

import (
	"context"
	"testing"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomAccountDetail(t *testing.T, accountID int64) AccountDetail {
	arg := CreateAccountDetailParams{
		AccountID: accountID,
		Type:      "IBAN",
		Value:     "MA" + utils.RandomNumbers(25),
		Country: pgtype.Text{
			String: "MA",
			Valid:  true,
		},
		BankName: pgtype.Text{
			String: "Attijariwafa Bank",
			Valid:  true,
		},
	}

	detail, err := testQueries.CreateAccountDetail(context.Background(), arg)
	require.NoError(t, err)

	require.NotEmpty(t, detail)
	require.Equal(t, accountID, detail.AccountID)
	require.Equal(t, arg.Type, detail.Type)
	require.Equal(t, arg.Value, detail.Value)
	require.Equal(t, arg.Country, detail.Country)
	require.Equal(t, arg.BankName, detail.BankName)

	return detail
}

func TestCreateAccountDetail(t *testing.T) {
	account := createRandomAccount(t)

	detail := createRandomAccountDetail(t, account.ID)

	require.NotZero(t, detail.ID)
	require.Equal(t, account.ID, detail.AccountID)
	require.Equal(t, "IBAN", detail.Type)

	require.True(t, detail.Country.Valid)
	require.Equal(t, "MA", detail.Country.String)

	require.True(t, detail.BankName.Valid)
	require.Equal(t, "Attijariwafa Bank", detail.BankName.String)
}

func TestGetAccountDetailByID(t *testing.T) {
	account := createRandomAccount(t)

	detail := createRandomAccountDetail(t, account.ID)

	result, err := testQueries.GetAccountDetailById(
		context.Background(),
		detail.ID,
	)

	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, detail.ID, result.ID)
	require.Equal(t, detail.AccountID, result.AccountID)
	require.Equal(t, detail.Type, result.Type)
	require.Equal(t, detail.Value, result.Value)

	require.Equal(t, detail.Country, result.Country)
	require.Equal(t, detail.BankName, result.BankName)
}

func TestGetAccountDetailByAccountID(t *testing.T) {
	account := createRandomAccount(t)

	detail := createRandomAccountDetail(t, account.ID)

	result, err := testQueries.GetAccountDetailByAccountId(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.NotEmpty(t, result)

	require.Equal(t, detail.ID, result.ID)
	require.Equal(t, account.ID, result.AccountID)
	require.Equal(t, detail.Type, result.Type)
	require.Equal(t, detail.Value, result.Value)

	require.True(t, result.Country.Valid)
	require.Equal(t, "MA", result.Country.String)
}

func TestListAccountDetails(t *testing.T) {
	account := createRandomAccount(t)

	detail1 := createRandomAccountDetail(t, account.ID)

	detail2 := createRandomAccountDetail(t, account.ID)

	arg := ListAccountDetailsParams{
		Limit:  10,
		Offset: 0,
	}

	details, err := testQueries.ListAccountDetails(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(details), 2)

	require.Contains(t, details, detail1)
	require.Contains(t, details, detail2)
}

func TestUpdateAccountDetail(t *testing.T) {
	account := createRandomAccount(t)

	detail := createRandomAccountDetail(t, account.ID)

	newAccount := createRandomAccount(t)

	arg := UpdateAccountDetailParams{
		ID:        detail.ID,
		AccountID: newAccount.ID,
		Type:      "ACCOUNT_NUMBER",
		Value:     "9876543210",
		Country: pgtype.Text{
			String: "FR",
			Valid:  true,
		},
		BankName: pgtype.Text{
			String: "BNP Paribas",
			Valid:  true,
		},
	}

	err := testQueries.UpdateAccountDetail(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	updated, err := testQueries.GetAccountDetailById(
		context.Background(),
		detail.ID,
	)

	require.NoError(t, err)

	require.Equal(t, arg.ID, updated.ID)
	require.Equal(t, arg.AccountID, updated.AccountID)
	require.Equal(t, arg.Type, updated.Type)
	require.Equal(t, arg.Value, updated.Value)

	require.True(t, updated.Country.Valid)
	require.Equal(t, "FR", updated.Country.String)

	require.True(t, updated.BankName.Valid)
	require.Equal(t, "BNP Paribas", updated.BankName.String)
}

func TestDeleteAccountDetail(t *testing.T) {
	account := createRandomAccount(t)

	detail := createRandomAccountDetail(t, account.ID)

	err := testQueries.DeleteAccountDetail(
		context.Background(),
		detail.ID,
	)

	require.NoError(t, err)

	_, err = testQueries.GetAccountDetailById(
		context.Background(),
		detail.ID,
	)

	require.Error(t, err)
}

func TestCreateAccountDetailNonexistentAccount(t *testing.T) {
	arg := CreateAccountDetailParams{
		AccountID: 999999999,
		Type:      "IBAN",
		Value:     "MA6401151900000123456789012",
		Country: pgtype.Text{
			String: "MA",
			Valid:  true,
		},
		BankName: pgtype.Text{
			String: "Attijariwafa Bank",
			Valid:  true,
		},
	}

	_, err := testQueries.CreateAccountDetail(
		context.Background(),
		arg,
	)

	require.Error(t, err)
}

func TestCreateAccountDetailDuplicateTypeValue(t *testing.T) {
	account := createRandomAccount(t)

	arg := CreateAccountDetailParams{
		AccountID: account.ID,
		Type:      "IBAN",
		Value:     "MA6401151900000123456789012",
		Country: pgtype.Text{
			String: "MA",
			Valid:  true,
		},
		BankName: pgtype.Text{
			String: "Attijariwafa Bank",
			Valid:  true,
		},
	}

	_, err := testQueries.CreateAccountDetail(
		context.Background(),
		arg,
	)

	require.NoError(t, err)

	_, err = testQueries.CreateAccountDetail(
		context.Background(),
		arg,
	)

	require.Error(t, err)
}
