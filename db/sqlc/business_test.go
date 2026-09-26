package db

import (
	"context"
	"testing"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)



func TestCreateBusinessTx(t *testing.T) {
	store := NewStore(testDb)

	user := createNewUser(t)

	errs := make(chan error, 4)
	results := make(chan CreateBusinessTxResult, 4)

	n := 4

	for i := 0; i < n; i++ {
		go func() {

			args := CreateBusinessParams{
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
		  _ ,err = store.GetBusinessMember(context.Background(), result.BusinessMember.ID)
		require.NoError(t ,err)
	}
}

func createNewBusiness(t *testing.T) Business {
	user := createNewUser(t)

	business, err := testQueries.CreateBusiness(
		context.Background(),
		CreateBusinessParams{
			OwnerID: user.ID,
			Name:    utils.RandomString(10),
			RegistrationNumber: pgtype.Text{
				String: utils.RandomString(10),
				Valid:  true,
			},
			Country: "MA",
			Status:  "active",
		},
	)

	require.NoError(t, err)

	return business
}

func TestCreateBusiness(t *testing.T) {
	business := createNewBusiness(t)

	require.NotZero(t, business.ID)
	require.NotZero(t, business.OwnerID)

	require.NotEmpty(t, business.Name)

	require.True(t, business.RegistrationNumber.Valid)
	require.NotEmpty(t, business.RegistrationNumber.String)

	require.Equal(t, "MA", business.Country)
	require.Equal(t, "active", business.Status)
}

func TestGetBusiness(t *testing.T) {
	createdBusiness := createNewBusiness(t)

	business, err := testQueries.GetBusiness(
		context.Background(),
		createdBusiness.ID,
	)

	require.NoError(t, err)

	require.Equal(t, createdBusiness.ID, business.ID)
	require.Equal(t, createdBusiness.OwnerID, business.OwnerID)
	require.Equal(t, createdBusiness.Name, business.Name)
	require.Equal(
		t,
		createdBusiness.RegistrationNumber,
		business.RegistrationNumber,
	)
	require.Equal(t, createdBusiness.Country, business.Country)
	require.Equal(t, createdBusiness.Status, business.Status)
}

func TestListBusinessesByOwnerID(t *testing.T) {
	user := createNewUser(t)

	_, err := testQueries.CreateBusiness(
		context.Background(),
		CreateBusinessParams{
			OwnerID: user.ID,
			Name:    utils.RandomString(10),
			RegistrationNumber: pgtype.Text{
				String: utils.RandomString(10),
				Valid:  true,
			},
			Country: "MA",
			Status:  "active",
		},
	)
	require.NoError(t, err)

	_, err = testQueries.CreateBusiness(
		context.Background(),
		CreateBusinessParams{
			OwnerID: user.ID,
			Name:    utils.RandomString(10),
			RegistrationNumber: pgtype.Text{
				String: utils.RandomString(10),
				Valid:  true,
			},
			Country: "MA",
			Status:  "active",
		},
	)
	require.NoError(t, err)

	businesses, err := testQueries.ListBusinessesByOwnerID(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, businesses, 2)

	require.Equal(t, user.ID, businesses[0].OwnerID)
	require.Equal(t, user.ID, businesses[1].OwnerID)
}

func TestUpdateBusiness(t *testing.T) {
	business := createNewBusiness(t)

	updatedName := utils.RandomString(12)
	updatedRegistrationNumber := utils.RandomString(12)

	updatedBusiness, err := testQueries.UpdateBusiness(
		context.Background(),
		UpdateBusinessParams{
			ID:   business.ID,
			Name: updatedName,
			RegistrationNumber: pgtype.Text{
				String: updatedRegistrationNumber,
				Valid:  true,
			},
			Country: "FR",
		},
	)

	require.NoError(t, err)

	require.Equal(t, business.ID, updatedBusiness.ID)
	require.Equal(t, business.OwnerID, updatedBusiness.OwnerID)

	require.Equal(t, updatedName, updatedBusiness.Name)

	require.True(t, updatedBusiness.RegistrationNumber.Valid)
	require.Equal(
		t,
		updatedRegistrationNumber,
		updatedBusiness.RegistrationNumber.String,
	)

	require.Equal(t, "FR", updatedBusiness.Country)

	// UpdateBusiness should not change the status.
	require.Equal(t, "active", updatedBusiness.Status)
}

func TestSuspendBusiness(t *testing.T) {
	business := createNewBusiness(t)

	updatedBusiness, err := testQueries.UpdateBusinessStatus(
		context.Background(),
		UpdateBusinessStatusParams{
			ID:     business.ID,
			Status: "suspended",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "suspended", updatedBusiness.Status)
}

func TestCloseBusiness(t *testing.T) {
	business := createNewBusiness(t)

	updatedBusiness, err := testQueries.UpdateBusinessStatus(
		context.Background(),
		UpdateBusinessStatusParams{
			ID:     business.ID,
			Status: "closed",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "closed", updatedBusiness.Status)
}

func TestCreateBusinessInvalidOwner(t *testing.T) {
	_, err := testQueries.CreateBusiness(
		context.Background(),
		CreateBusinessParams{
			OwnerID: 999999999,
			Name:    utils.RandomString(10),
			RegistrationNumber: pgtype.Text{
				String: utils.RandomString(10),
				Valid:  true,
			},
			Country: "MA",
			Status:  "active",
		},
	)

	require.Error(t, err)
}