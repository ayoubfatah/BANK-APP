package db

import (
	"context"
	"database/sql"
	"testing"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createNewUser(t *testing.T) User {
	arg := CreateUserParams{
		FirstName: utils.RandomFirstName(),
		LastName:  utils.RandomLastName(),
		Email:     utils.RandomEmail(),
		Phone: pgtype.Text{
			String: utils.RandomPhone(),
			Valid:  true,
		},
		Status:       utils.RandomStatus(),
		PasswordHash: utils.RandomPasswordHash(),
	}

	user, err := testQueries.CreateUser(context.Background(), arg)

	require.NoError(t, err)
	require.NotEmpty(t, user)

	require.Equal(t, arg.FirstName, user.FirstName)
	require.Equal(t, arg.LastName, user.LastName)
	require.Equal(t, arg.Email, user.Email)
	require.Equal(t, arg.Phone, user.Phone)
	require.Equal(t, arg.Status, user.Status)
	require.Equal(t, arg.PasswordHash, user.PasswordHash)

	require.NotZero(t, user.ID)
	require.NotZero(t, user.CreatedAt)
	require.NotZero(t, user.UpdatedAt)

	return user
}

func requireUserFieldsEqual(t *testing.T, expected, actual User) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.FirstName, actual.FirstName)
	require.Equal(t, expected.LastName, actual.LastName)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.Phone, actual.Phone)
	require.Equal(t, expected.PasswordHash, actual.PasswordHash)
	require.Equal(t, expected.Status, actual.Status)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
}

// CREATE

func TestCreateUser(t *testing.T) {
	createNewUser(t)
}

// GET

func TestGetUser(t *testing.T) {
	newCreatedUser := createNewUser(t)

	user, err := testQueries.GetUser(
		context.Background(),
		newCreatedUser.ID,
	)

	require.NoError(t, err)

	requireUserFieldsEqual(t, newCreatedUser, user)
	require.Equal(t, newCreatedUser.UpdatedAt, user.UpdatedAt)
}

func TestGetUserNotFound(t *testing.T) {
	_, err := testQueries.GetUser(
		context.Background(),
		999999,
	)

	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

// DELETE

func TestDeleteUser(t *testing.T) {
	newCreatedUser := createNewUser(t)

	err := testQueries.DeleteUser(
		context.Background(),
		newCreatedUser.ID,
	)

	require.NoError(t, err)

	_, err = testQueries.GetUser(
		context.Background(),
		newCreatedUser.ID,
	)

	require.Error(t, err)
	require.ErrorIs(t, err, sql.ErrNoRows)
}

func TestDeleteUserNotFound(t *testing.T) {
	err := testQueries.DeleteUser(
		context.Background(),
		999999,
	)

	require.NoError(t, err)
}

// UPDATE USER

func TestUpdateUser(t *testing.T) {
	newCreatedUser := createNewUser(t)

	args := UpdateUserParams{
		ID:        newCreatedUser.ID,
		FirstName: "ayoub",
		LastName:  "fatah",
		Img: pgtype.Text{
			String: "dsfsdfsfsdf",
			Valid: true,
		},
	}

	newUpdatedUser, err := testQueries.UpdateUser(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotEmpty(t, newUpdatedUser)

	require.Equal(t, newCreatedUser.ID, newUpdatedUser.ID)
	require.Equal(t, newCreatedUser.Email, newUpdatedUser.Email)
	require.Equal(t, newCreatedUser.PasswordHash, newUpdatedUser.PasswordHash)
	require.Equal(t, newCreatedUser.Status, newUpdatedUser.Status)
	require.Equal(t, newCreatedUser.Phone, newUpdatedUser.Phone)
	require.Equal(t, newCreatedUser.CreatedAt, newUpdatedUser.CreatedAt)

	require.NotEqual(t, newCreatedUser.FirstName, newUpdatedUser.FirstName)
	require.NotEqual(t, newCreatedUser.LastName, newUpdatedUser.LastName)

	require.Equal(t, args.FirstName, newUpdatedUser.FirstName)
	require.Equal(t, args.LastName, newUpdatedUser.LastName)

	require.NotEqual(t, newCreatedUser.UpdatedAt, newUpdatedUser.UpdatedAt)
}

func TestUpdateUserNotFound(t *testing.T) {
	args := UpdateUserParams{
		ID:        999999,
		FirstName: "ayoub",
		LastName:  "fatah",
	}

	_, err := testQueries.UpdateUser(
		context.Background(),
		args,
	)

	require.ErrorIs(t, err, sql.ErrNoRows)
}

// UPDATE EMAIL

func TestUpdateUserEmail(t *testing.T) {
	newCreatedUser := createNewUser(t)

	args := UpdateUserEmailParams{
		ID:    newCreatedUser.ID,
		Email: utils.RandomEmail(),
	}

	updatedUser, err := testQueries.UpdateUserEmail(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	require.Equal(t, newCreatedUser.ID, updatedUser.ID)
	require.Equal(t, newCreatedUser.FirstName, updatedUser.FirstName)
	require.Equal(t, newCreatedUser.LastName, updatedUser.LastName)
	require.Equal(t, newCreatedUser.Phone, updatedUser.Phone)
	require.Equal(t, newCreatedUser.PasswordHash, updatedUser.PasswordHash)
	require.Equal(t, newCreatedUser.Status, updatedUser.Status)
	require.Equal(t, newCreatedUser.CreatedAt, updatedUser.CreatedAt)

	require.NotEqual(t, newCreatedUser.Email, updatedUser.Email)
	require.Equal(t, args.Email, updatedUser.Email)

	require.NotEqual(t, newCreatedUser.UpdatedAt, updatedUser.UpdatedAt)
}

func TestUpdateUserEmailNotFound(t *testing.T) {
	args := UpdateUserEmailParams{
		ID:    999999,
		Email: utils.RandomEmail(),
	}

	_, err := testQueries.UpdateUserEmail(
		context.Background(),
		args,
	)

	require.ErrorIs(t, err, sql.ErrNoRows)
}

// UPDATE PASSWORD

func TestUpdateUserPassword(t *testing.T) {
	newCreatedUser := createNewUser(t)

	args := UpdateUserPasswordParams{
		ID:           newCreatedUser.ID,
		PasswordHash: utils.RandomPasswordHash(),
	}

	err := testQueries.UpdateUserPassword(
		context.Background(),
		args,
	)

	require.NoError(t, err)

	updatedUser, err := testQueries.GetUser(
		context.Background(),
		newCreatedUser.ID,
	)

	require.NoError(t, err)

	require.Equal(t, newCreatedUser.ID, updatedUser.ID)
	require.Equal(t, newCreatedUser.FirstName, updatedUser.FirstName)
	require.Equal(t, newCreatedUser.LastName, updatedUser.LastName)
	require.Equal(t, newCreatedUser.Email, updatedUser.Email)
	require.Equal(t, newCreatedUser.Phone, updatedUser.Phone)
	require.Equal(t, newCreatedUser.Status, updatedUser.Status)
	require.Equal(t, newCreatedUser.CreatedAt, updatedUser.CreatedAt)

	require.NotEqual(t, newCreatedUser.PasswordHash, updatedUser.PasswordHash)
	require.Equal(t, args.PasswordHash, updatedUser.PasswordHash)

	require.NotEqual(t, newCreatedUser.UpdatedAt, updatedUser.UpdatedAt)
}

func TestUpdateUserPasswordNotFound(t *testing.T) {
	args := UpdateUserPasswordParams{
		ID:           999999,
		PasswordHash: utils.RandomPasswordHash(),
	}

	err := testQueries.UpdateUserPassword(
		context.Background(),
		args,
	)

	require.NoError(t, err)
}

// UPDATE PHONE

func TestUpdateUserPhone(t *testing.T) {
	newCreatedUser := createNewUser(t)

	args := UpdateUserPhoneParams{
		ID: newCreatedUser.ID,
		Phone: pgtype.Text{
			String: utils.RandomPhone(),
			Valid:  true,
		},
	}

	err := testQueries.UpdateUserPhone(
		context.Background(),
		args,
	)

	require.NoError(t, err)

	updatedUser, err := testQueries.GetUser(
		context.Background(),
		newCreatedUser.ID,
	)

	require.NoError(t, err)

	require.Equal(t, newCreatedUser.ID, updatedUser.ID)
	require.Equal(t, newCreatedUser.FirstName, updatedUser.FirstName)
	require.Equal(t, newCreatedUser.LastName, updatedUser.LastName)
	require.Equal(t, newCreatedUser.Email, updatedUser.Email)
	require.Equal(t, newCreatedUser.PasswordHash, updatedUser.PasswordHash)
	require.Equal(t, newCreatedUser.Status, updatedUser.Status)
	require.Equal(t, newCreatedUser.CreatedAt, updatedUser.CreatedAt)

	require.NotEqual(t, newCreatedUser.Phone, updatedUser.Phone)
	require.Equal(t, args.Phone, updatedUser.Phone)

	require.NotEqual(t, newCreatedUser.UpdatedAt, updatedUser.UpdatedAt)
}

func TestUpdateUserPhoneNotFound(t *testing.T) {
	args := UpdateUserPhoneParams{
		ID: 999999,
		Phone: pgtype.Text{
			String: utils.RandomPhone(),
			Valid:  true,
		},
	}

	err := testQueries.UpdateUserPhone(
		context.Background(),
		args,
	)

	require.NoError(t, err)
}

// UPDATE STATUS

func TestUpdateUserStatus(t *testing.T) {
	newCreatedUser := createNewUser(t)

	args := UpdateUserStatusParams{
		ID:     newCreatedUser.ID,
		Status: "CEO",
	}

	updatedUser, err := testQueries.UpdateUserStatus(
		context.Background(),
		args,
	)

	require.NoError(t, err)
	require.NotEmpty(t, updatedUser)

	require.Equal(t, newCreatedUser.ID, updatedUser.ID)
	require.Equal(t, newCreatedUser.FirstName, updatedUser.FirstName)
	require.Equal(t, newCreatedUser.LastName, updatedUser.LastName)
	require.Equal(t, newCreatedUser.Email, updatedUser.Email)
	require.Equal(t, newCreatedUser.Phone, updatedUser.Phone)
	require.Equal(t, newCreatedUser.PasswordHash, updatedUser.PasswordHash)
	require.Equal(t, newCreatedUser.CreatedAt, updatedUser.CreatedAt)

	require.NotEqual(t, newCreatedUser.Status, updatedUser.Status)
	require.Equal(t, args.Status, updatedUser.Status)

	require.NotEqual(t, newCreatedUser.UpdatedAt, updatedUser.UpdatedAt)
}

func TestUpdateUserStatusNotFound(t *testing.T) {
	args := UpdateUserStatusParams{
		ID:     999999,
		Status: utils.RandomStatus(),
	}

	_, err := testQueries.UpdateUserStatus(
		context.Background(),
		args,
	)

	require.ErrorIs(t, err, sql.ErrNoRows)
}


