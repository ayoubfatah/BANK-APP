package db

import (
	"context"
	"testing"
	"time"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createNewPaymentLink(t *testing.T) PaymentLink {
	business := createNewBusiness(t)

	amount := int64(10000)
	currency := "USD"
	description := utils.RandomString(20)
	token := utils.RandomString(32)

	paymentLink, err := testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      token,
			Amount: pgtype.Int8{
				Int64: amount,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: currency,
				Valid:  true,
			},
			Description: pgtype.Text{
				String: description,
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		},
	)

	require.NoError(t, err)

	return paymentLink
}

func TestCreatePaymentLink(t *testing.T) {
	paymentLink := createNewPaymentLink(t)

	require.NotZero(t, paymentLink.ID)
	require.NotZero(t, paymentLink.BusinessID)

	require.NotEmpty(t, paymentLink.Token)
	require.True(t, paymentLink.Amount.Valid)
	require.Equal(t, int64(10000), paymentLink.Amount.Int64)

	require.True(t, paymentLink.Currency.Valid)
	require.Equal(t, "USD", paymentLink.Currency.String)

	require.True(t, paymentLink.Description.Valid)
	require.NotEmpty(t, paymentLink.Description.String)

	require.Equal(t, "active", paymentLink.Status)

	require.True(t, paymentLink.ExpiresAt.Valid)
	require.False(t, paymentLink.ExpiresAt.Time.IsZero())
}

func TestGetPaymentLink(t *testing.T) {
	createdPaymentLink := createNewPaymentLink(t)

	paymentLink, err := testQueries.GetPaymentLink(
		context.Background(),
		createdPaymentLink.ID,
	)

	require.NoError(t, err)

	require.Equal(t, createdPaymentLink.ID, paymentLink.ID)
	require.Equal(t, createdPaymentLink.BusinessID, paymentLink.BusinessID)
	require.Equal(t, createdPaymentLink.Token, paymentLink.Token)
	require.Equal(t, createdPaymentLink.Amount, paymentLink.Amount)
	require.Equal(t, createdPaymentLink.Currency, paymentLink.Currency)
	require.Equal(t, createdPaymentLink.Description, paymentLink.Description)
	require.Equal(t, createdPaymentLink.Status, paymentLink.Status)
	require.Equal(t, createdPaymentLink.ExpiresAt, paymentLink.ExpiresAt)
}

func TestGetPaymentLinkByToken(t *testing.T) {
	createdPaymentLink := createNewPaymentLink(t)

	paymentLink, err := testQueries.GetPaymentLinkByToken(
		context.Background(),
		createdPaymentLink.Token,
	)

	require.NoError(t, err)

	require.Equal(t, createdPaymentLink.ID, paymentLink.ID)
	require.Equal(t, createdPaymentLink.Token, paymentLink.Token)
	require.Equal(t, createdPaymentLink.BusinessID, paymentLink.BusinessID)
}

func TestPaymentLinkTokenIsUnique(t *testing.T) {
	business := createNewBusiness(t)

	token := utils.RandomString(32)

	_, err := testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      token,
			Amount: pgtype.Int8{
				Int64: 5000,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: "USD",
				Valid:  true,
			},
			Description: pgtype.Text{
				String: utils.RandomString(20),
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		},
	)

	require.NoError(t, err)

	_, err = testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      token,
			Amount: pgtype.Int8{
				Int64: 7000,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: "USD",
				Valid:  true,
			},
			Description: pgtype.Text{
				String: utils.RandomString(20),
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		},
	)

	require.Error(t, err)
}

func TestListPaymentLinksByBusinessID(t *testing.T) {
	business := createNewBusiness(t)

	_, err := testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      utils.RandomString(32),
			Amount: pgtype.Int8{
				Int64: 1000,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: "USD",
				Valid:  true,
			},
			Description: pgtype.Text{
				String: utils.RandomString(20),
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	_, err = testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      utils.RandomString(32),
			Amount: pgtype.Int8{
				Int64: 2000,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: "USD",
				Valid:  true,
			},
			Description: pgtype.Text{
				String: utils.RandomString(20),
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(24 * time.Hour),
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	links, err := testQueries.ListPaymentLinksByBusinessID(
		context.Background(),
		business.ID,
	)

	require.NoError(t, err)
	require.Len(t, links, 2)

	require.Equal(t, business.ID, links[0].BusinessID)
	require.Equal(t, business.ID, links[1].BusinessID)
}

func TestMarkPaymentLinkAsPaid(t *testing.T) {
	paymentLink := createNewPaymentLink(t)

	updatedLink, err := testQueries.UpdatePaymentLinkStatus(
		context.Background(),
		UpdatePaymentLinkStatusParams{
			ID:     paymentLink.ID,
			Status: "paid",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "paid", updatedLink.Status)
}

func TestExpirePaymentLink(t *testing.T) {
	paymentLink := createNewPaymentLink(t)

	updatedLink, err := testQueries.UpdatePaymentLinkStatus(
		context.Background(),
		UpdatePaymentLinkStatusParams{
			ID:     paymentLink.ID,
			Status: "expired",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "expired", updatedLink.Status)
}

func TestCancelPaymentLink(t *testing.T) {
	paymentLink := createNewPaymentLink(t)

	updatedLink, err := testQueries.UpdatePaymentLinkStatus(
		context.Background(),
		UpdatePaymentLinkStatusParams{
			ID:     paymentLink.ID,
			Status: "cancelled",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "cancelled", updatedLink.Status)
}

func TestExpiredPaymentLinkCannotBePaid(t *testing.T) {
	business := createNewBusiness(t)

	paymentLink, err := testQueries.CreatePaymentLink(
		context.Background(),
		CreatePaymentLinkParams{
			BusinessID: business.ID,
			Token:      utils.RandomString(32),
			Amount: pgtype.Int8{
				Int64: 10000,
				Valid: true,
			},
			Currency: pgtype.Text{
				String: "USD",
				Valid:  true,
			},
			Description: pgtype.Text{
				String: utils.RandomString(20),
				Valid:  true,
			},
			Status: "active",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().Add(-1 * time.Hour),
				Valid: true,
			},
		},
	)

	require.NoError(t, err)

	_, err = testQueries.UpdatePaymentLinkStatus(
		context.Background(),
		UpdatePaymentLinkStatusParams{
			ID:     paymentLink.ID,
			Status: "paid",
		},
	)

	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestDeletePaymentLink(t *testing.T) {
	paymentLink := createNewPaymentLink(t)

	err := testQueries.DeletePaymentLink(
		context.Background(),
		paymentLink.ID,
	)

	require.NoError(t, err)

	_, err = testQueries.GetPaymentLink(
		context.Background(),
		paymentLink.ID,
	)

	require.ErrorIs(t, err, pgx.ErrNoRows)
}
