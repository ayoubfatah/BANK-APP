package store

import (
	"context"
	"testing"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createRandomCard(t *testing.T) db.Card {
	t.Helper()

	user := createNewUser(t)

	account, err := testStore.CreateAccount(
		context.Background(),
		db.CreateAccountParams{
			UserID:   user.ID,
			Currency: "USD",
			Balance:  100000,
			Status:   "active",
		},
	)
	require.NoError(t, err)

	card, err := testStore.CreateCard(
		context.Background(),
		db.CreateCardParams{
			UserID:   user.ID,
			AccountID: account.ID,
			LastFour: "4242",
			CardType: "physical",
			Status:   "active",
			ExpiryDate: pgtype.Date{
				Time:  time.Now().AddDate(3, 0, 0),
				Valid: true,
			},
		},
	)
	require.NoError(t, err)

	return card
}

func createCardTransaction(t *testing.T) db.CardTransaction {
	store := NewStore(testStore.db)

	card := createRandomCard(t)

	account, err := store.GetAccount(
		context.Background(),
		card.AccountID,
	)
	require.NoError(t, err)

	arg := CreateCardTransactionParams{
		CardID:           card.ID,
		AccountID:        account.ID,
		MerchantName:     "Starbucks",
		MerchantCategory: pgtype.Text{String: "coffee_shop", Valid: true},
		Amount:           450,
		Currency:         account.Currency,
		AuthorizationCode: pgtype.Text{
			String: "AUTH123456",
			Valid:  true,
		},
	}

	cardTxn, err := store.CreateCardTransaction(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.NotEmpty(t, cardTxn)

	require.Equal(t, arg.CardID, cardTxn.CardID)
	require.Equal(t, arg.AccountID, cardTxn.AccountID)
	require.Equal(t, arg.MerchantName, cardTxn.MerchantName)
	require.Equal(t, arg.MerchantCategory, cardTxn.MerchantCategory)
	require.Equal(t, arg.Amount, cardTxn.Amount)
	require.Equal(t, arg.Currency, cardTxn.Currency)
	require.Equal(t, arg.AuthorizationCode, cardTxn.AuthorizationCode)

	require.Equal(t, "authorized", cardTxn.Status)
	require.NotZero(t, cardTxn.ID)
	require.NotZero(t, cardTxn.CreatedAt)
	return cardTxn
}


func TestCreateCardTransaction(t *testing.T) {
	createCardTransaction(t)
}

func TestCaptureCardTransactionTx(t *testing.T) {
	store := NewStore(testStore.db)

	// Create a card and its account
	card := createRandomCard(t)

	account, err := store.GetAccount(
		context.Background(),
		card.AccountID,
	)
	require.NoError(t, err)

	originalBalance := account.Balance

	// Create an authorized card transaction
	cardTxn, err := store.CreateCardTransaction(
		context.Background(),
		CreateCardTransactionParams{
			CardID:           card.ID,
			AccountID:        account.ID,
			MerchantName:     "Starbucks",
			MerchantCategory: pgtype.Text{
				String: "coffee_shop",
				Valid:  true,
			},
			Amount:   500,
			Currency: account.Currency,
			AuthorizationCode: pgtype.Text{
				String: "AUTH123456",
				Valid:  true,
			},
		},
	)

	require.NoError(t, err)

	// It should start as authorized
	require.Equal(t, "authorized", cardTxn.Status)
	require.Equal(t, int64(500), cardTxn.Amount)

	// Final amount can be different from the authorized amount
	finalAmount := int64(450)

	// Capture the transaction
	result, err := store.CaptureCardTransactionTx(
		context.Background(),
		cardTxn.ID,
		finalAmount,
	)

	require.NoError(t, err)

	// --------------------------------------------------
	// 1. Verify card transaction
	// --------------------------------------------------

	require.Equal(t, cardTxn.ID, result.ID)
	require.Equal(t, cardTxn.CardID, result.CardID)
	require.Equal(t, cardTxn.AccountID, result.AccountID)

	// Status should change from authorized -> captured
	require.Equal(t, "captured", result.Status)

	// Amount should change to the final amount
	require.Equal(t, finalAmount, result.Amount)

	// Currency should stay the same
	require.Equal(t, account.Currency, result.Currency)

	// --------------------------------------------------
	// 2. Verify entry
	// --------------------------------------------------

	entries, err := store.GetEntriesByReferenceID(
	context.Background(),
	pgtype.Int8{
		Int64: cardTxn.ID,
		Valid: true,
	},
)

	require.NoError(t, err)
	require.Len(t, entries, 1)

	entry := entries[0]

	require.NoError(t, err)

	require.Equal(t, account.ID, entry.AccountID)
	require.Equal(t, finalAmount, entry.Amount)
	require.Equal(t, account.Currency, entry.Currency)
	require.Equal(t, "card", entry.Type)

	require.Equal(
		t,
		pgtype.Int8{
			Int64: cardTxn.ID,
			Valid: true,
		},
		entry.ReferenceID,
	)

	// --------------------------------------------------
	// 3. Verify account balance
	// --------------------------------------------------

	expectedBalance := originalBalance - finalAmount

	updatedAccount, err := store.GetAccount(
		context.Background(),
		account.ID,
	)

	require.NoError(t, err)

	require.Equal(
		t,
		expectedBalance,
		updatedAccount.Balance,
	)

	// --------------------------------------------------
	// 4. Verify transaction directly from database
	// --------------------------------------------------

	updatedCardTxn, err := store.GetCardTransaction(
		context.Background(),
		cardTxn.ID,
	)

	require.NoError(t, err)

	require.Equal(t, "captured", updatedCardTxn.Status)
	require.Equal(t, finalAmount, updatedCardTxn.Amount)
	require.Equal(t, account.ID, updatedCardTxn.AccountID)
}



func TestVoidCardTransaction(t *testing.T) {
	store := NewStore(testStore.db)

	txn := createCardTransaction(t)

	voided, err := store.VoidCardTransaction(context.Background(), txn.ID)

	require.NoError(t, err)
	require.Equal(t, txn.ID, voided.ID)
	require.Equal(t, "voided", voided.Status)
	require.Equal(t, txn.Amount, voided.Amount) // amount untouched
}

func TestVoidCardTransaction_AlreadyVoided(t *testing.T) {
	store := NewStore(testStore.db)
	txn := createCardTransaction(t)

	_, err := store.VoidCardTransaction(context.Background(), txn.ID)
	require.NoError(t, err)

	// second void: WHERE status = 'authorized' matches nothing
	_, err = store.VoidCardTransaction(context.Background(), txn.ID)
	require.ErrorIs(t, err, pgx.ErrNoRows)
}

func TestExpireOldAuthorizations(t *testing.T) {
	store := NewStore(testStore.db)
	txn := createCardTransaction(t)

	// maxAge of 0 => everything authorized before "now" is stale
	expired, err := store.ExpireOldAuthorizations(context.Background(), pgtype.Interval{Valid: true})
	require.NoError(t, err)

	var found bool
	for _, e := range expired {
		require.Equal(t, "expired", e.Status)
		if e.ID == txn.ID {
			found = true 
		}
	}
	require.True(t, found)
}