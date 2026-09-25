package db

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createNewCard(t *testing.T) Card {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		UserID:   user.ID,
		Currency: "USD",
		Balance:  100000,
		Status:   "active",
	})
	require.NoError(t, err)

	card, err := testQueries.CreateCard(context.Background(), CreateCardParams{
		UserID:    user.ID,
		AccountID: account.ID,
		LastFour:  "4242",
		CardType:  "physical",
		Status:    "active",
		ExpiryDate:  pgtype.Date{
			Time: time.Now().AddDate(3, 0, 0),
			Valid: true,
		},
	})
	require.NoError(t, err)

	return card
}

func TestCreateCard(t *testing.T) {
	card := createNewCard(t)

	require.NotZero(t, card.ID)
	require.NotZero(t, card.UserID)
	require.NotZero(t, card.AccountID)

	require.Equal(t, "4242", card.LastFour)
	require.Equal(t, "physical", card.CardType)
	require.Equal(t, "active", card.Status)
	require.False(t, card.ExpiryDate.Time.IsZero())
}

func TestGetCard(t *testing.T) {
	createdCard := createNewCard(t)

	card, err := testQueries.GetCard(
		context.Background(),
		createdCard.ID,
	)

	require.NoError(t, err)

	require.Equal(t, createdCard.ID, card.ID)
	require.Equal(t, createdCard.UserID, card.UserID)
	require.Equal(t, createdCard.AccountID, card.AccountID)
	require.Equal(t, createdCard.LastFour, card.LastFour)
	require.Equal(t, createdCard.CardType, card.CardType)
	require.Equal(t, createdCard.Status, card.Status)
	require.Equal(t, createdCard.ExpiryDate, card.ExpiryDate)
}

func TestListCardsByUserID(t *testing.T) {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		UserID:   user.ID,
		Currency: "USD",
		Balance:  100000,
		Status:   "active",
	})
	require.NoError(t, err)

	_, err = testQueries.CreateCard(context.Background(), CreateCardParams{
		UserID:     user.ID,
		AccountID:  account.ID,
		LastFour:   "1111",
		CardType:   "physical",
		Status:     "active",
		ExpiryDate:  pgtype.Date{
			Time: time.Now().AddDate(3, 0, 0),
			Valid: true,
		},
	})
	require.NoError(t, err)

	_, err = testQueries.CreateCard(context.Background(), CreateCardParams{
		UserID:     user.ID,
		AccountID:  account.ID,
		LastFour:   "2222",
		CardType:   "virtual",
		Status:     "active",
		ExpiryDate:  pgtype.Date{
			Time: time.Now().AddDate(3, 0, 0),
			Valid: true,
		},
	})
	require.NoError(t, err)

	cards, err := testQueries.ListCardsByUserID(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, cards, 2)

	require.Equal(t, user.ID, cards[0].UserID)
	require.Equal(t, user.ID, cards[1].UserID)
}

func TestListCardsByAccountID(t *testing.T) {
	user := createNewUser(t)

	account, err := testQueries.CreateAccount(context.Background(), CreateAccountParams{
		UserID:   user.ID,
		Currency: "USD",
		Balance:  100000,
		Status:   "active",
	})
	require.NoError(t, err)

	_, err = testQueries.CreateCard(context.Background(), CreateCardParams{
		UserID:     user.ID,
		AccountID:  account.ID,
		LastFour:   "3333",
		CardType:   "physical",
		Status:     "active",
		ExpiryDate:  pgtype.Date{
			Time: time.Now().AddDate(3, 0, 0),
			Valid: true,
		},
	})
	require.NoError(t, err)

	_, err = testQueries.CreateCard(context.Background(), CreateCardParams{
		UserID:     user.ID,
		AccountID:  account.ID,
		LastFour:   "4444",
		CardType:   "virtual",
		Status:     "active",
		ExpiryDate: pgtype.Date{
			Time: time.Now().AddDate(3, 0, 0),
			Valid: true,
		},
	})
	require.NoError(t, err)

	cards, err := testQueries.ListCardsByAccountID(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)
	require.Len(t, cards, 2)

	require.Equal(t, account.ID, cards[0].AccountID)
	require.Equal(t, account.ID, cards[1].AccountID)
}

func TestFreezeCard(t *testing.T) {
	card := createNewCard(t)

	updatedCard, err := testQueries.UpdateCardStatus(
		context.Background(),
		UpdateCardStatusParams{
			ID:     card.ID,
			Status: "frozen",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "frozen", updatedCard.Status)
}

func TestUnfreezeCard(t *testing.T) {
	card := createNewCard(t)

	_, err := testQueries.UpdateCardStatus(
		context.Background(),
		UpdateCardStatusParams{
			ID:     card.ID,
			Status: "frozen",
		},
	)
	require.NoError(t, err)

	updatedCard, err := testQueries.UpdateCardStatus(
		context.Background(),
		UpdateCardStatusParams{
			ID:     card.ID,
			Status: "active",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "active", updatedCard.Status)
}

func TestCancelCard(t *testing.T) {
	card := createNewCard(t)

	updatedCard, err := testQueries.UpdateCardStatus(
		context.Background(),
		UpdateCardStatusParams{
			ID:     card.ID,
			Status: "cancelled",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "cancelled", updatedCard.Status)
}

func TestExpireCard(t *testing.T) {
	card := createNewCard(t)

	updatedCard, err := testQueries.UpdateCardStatus(
		context.Background(),
		UpdateCardStatusParams{
			ID:     card.ID,
			Status: "expired",
		},
	)

	require.NoError(t, err)
	require.Equal(t, "expired", updatedCard.Status)
}

func TestCardBelongsToCorrectUser(t *testing.T) {
	card := createNewCard(t)

	cardFromDB, err := testQueries.GetCard(
		context.Background(),
		card.ID,
	)

	require.NoError(t, err)
	require.Equal(t, card.UserID, cardFromDB.UserID)
}

func TestCardBelongsToCorrectAccount(t *testing.T) {
	card := createNewCard(t)

	cardFromDB, err := testQueries.GetCard(
		context.Background(),
		card.ID,
	)

	require.NoError(t, err)
	require.Equal(t, card.AccountID, cardFromDB.AccountID)
}