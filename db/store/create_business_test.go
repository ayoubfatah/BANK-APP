package store

import (
	"context"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createNewUser(t *testing.T) db.User {
	arg := db.CreateUserParams{
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

	user, err := testStore.CreateUser(context.Background(), arg)

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

func TestCreateBusinessTx(t *testing.T) {
	store := testStore

	user := createNewUser(t)

	errs := make(chan error, 4)
	results := make(chan CreateBusinessTxResult, 4)

	n := 4

	for i := 0; i < n; i++ {
		go func() {

			args := db.CreateBusinessParams{
				OwnerID: user.ID,

				Name: "business ino",

				RegistrationNumber: pgtype.Text{
					String: "34234234234234",
					Valid:  true,
				},

				Country: "MA",
				Status:  "active",
			}

			result, err := store.CreateBusinessTx(
				context.Background(),
				args,
				"owner",
			)

			errs <- err
			results <- result

		}()
	}

	// check results
	for i := 0; i < n; i++ {

		err := <-errs
		require.NoError(t, err)

		result := <-results

		require.NotZero(t, result.Business.ID)
		require.Equal(t, user.ID, result.Business.OwnerID)
		require.Equal(t, "business ino", result.Business.Name)

		require.NotZero(t, result.BusinessMember.ID)

		require.Equal(
			t,
			result.Business.ID,
			result.BusinessMember.BusinessID,
		)

		require.Equal(
			t,
			user.ID,
			result.BusinessMember.UserID,
		)

		require.Equal(
			t,
			"owner",
			result.BusinessMember.Role,
		)
		_, err = store.GetBusinessMember(context.Background(), result.BusinessMember.ID)
		require.NoError(t, err)
	}
}
