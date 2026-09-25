package db

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func createRandomAccount(t *testing.T) Account {
	user := createNewUser(t)

	arg := CreateAccountParams{
		UserID:   user.ID,
		Currency: "USD",
		Status:   "active",
		Balance:  0,
	}

	account, err := testQueries.CreateAccount(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotZero(t, account.ID)

	return account
}

func TestCreateAccount(t *testing.T) {
	user := createNewUser(t)

	arg := CreateAccountParams{
		UserID:   user.ID,
		Currency: "USD",
		Status:   "active",
		Balance:  0,
	}

	account, err := testQueries.CreateAccount(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotZero(t, account.ID)

	require.Equal(t, user.ID, account.UserID)
	require.Equal(t, "USD", account.Currency)
	require.Equal(t, int64(0), account.Balance)
	require.Equal(t, "active", account.Status)
	require.NotZero(t, account.CreatedAt)
	require.NotZero(t, account.UpdatedAt)
}

func TestGetAccount(t *testing.T) {
	account := createRandomAccount(t)

	result, err := testQueries.GetAccount(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)

	require.Equal(t, account.ID, result.ID)
	require.Equal(t, account.UserID, result.UserID)
	require.Equal(t, account.Currency, result.Currency)
	require.Equal(t, account.Balance, result.Balance)
	require.Equal(t, account.Status, result.Status)
}

func TestGetAccountByUserAndCurrency(t *testing.T) {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	result, err := testQueries.GetAccountByUserAndCurrency(
		context.Background(),
		GetAccountByUserAndCurrencyParams{
			UserID:   user.ID,
			Currency: "USD",
		},
	)

	require.NoError(t, err)

	require.Equal(t, account.ID, result.ID)
	require.Equal(t, user.ID, result.UserID)
	require.Equal(t, "USD", result.Currency)
}

func TestListAccountsByUserID(t *testing.T) {
	user := createNewUser(t)

	usdAccount, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	eurAccount, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "EUR",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	accounts, err := testQueries.ListAccountsByUserID(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, accounts, 2)

	require.Equal(t, user.ID, accounts[0].UserID)
	require.Equal(t, user.ID, accounts[1].UserID)

	require.Contains(t, []int64{
		usdAccount.ID,
		eurAccount.ID,
	}, accounts[0].ID)

	require.Contains(t, []int64{
		usdAccount.ID,
		eurAccount.ID,
	}, accounts[1].ID)
}

func TestCreateUSDAccount(t *testing.T) {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	require.Equal(t, user.ID, account.UserID)
	require.Equal(t, "USD", account.Currency)
	require.Equal(t, "active", account.Status)
	require.Equal(t, int64(0), account.Balance)
}

func TestCreateEURAccount(t *testing.T) {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "EUR",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	require.Equal(t, user.ID, account.UserID)
	require.Equal(t, "EUR", account.Currency)
	require.Equal(t, "active", account.Status)
	require.Equal(t, int64(0), account.Balance)
}

func TestCreateAccount_DuplicateUserCurrency(t *testing.T) {
	user := createNewUser(t)

	_, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.NoError(t, err)

	_, err = testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.Error(t, err)
}

func TestUpdateAccountStatus_Freeze(t *testing.T) {
	account := createRandomAccount(t)

	updated, err := testQueries.UpdateAccountStatus(
		context.Background(),
		UpdateAccountStatusParams{
			ID:     account.ID,
			Status: "frozen",
		},
	)

	require.NoError(t, err)

	require.Equal(t, account.ID, updated.ID)
	require.Equal(t, account.UserID, updated.UserID)
	require.Equal(t, "frozen", updated.Status)
}

func TestUpdateAccountStatus_Close(t *testing.T) {
	account := createRandomAccount(t)

	updated, err := testQueries.UpdateAccountStatus(
		context.Background(),
		UpdateAccountStatusParams{
			ID:     account.ID,
			Status: "closed",
		},
	)

	require.NoError(t, err)

	require.Equal(t, account.ID, updated.ID)
	require.Equal(t, account.UserID, updated.UserID)
	require.Equal(t, "closed", updated.Status)
}

func TestCreateAccount_NonexistentUser(t *testing.T) {
	_, err := testQueries.CreateAccount(
		context.Background(),
		CreateAccountParams{
			UserID:   999999999,
			Currency: "USD",
			Status:   "active",
			Balance:  0,
		},
	)

	require.Error(t, err)
}