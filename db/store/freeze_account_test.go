package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFreezeAccountTxByUser(t *testing.T) {
	store := NewStore(testStore.db)
	ctx := context.Background()

	user := createNewUser(t)

	account := createTestAccount(t, store, user.ID, "USD", int64(3000))

	rowsAffected, err := testStore.FreezeAccountTx(ctx, FreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    user.ID,
		ActorID:   user.ID,
		ActorRole: user.Role,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	frozenAccount, err := testStore.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "frozen", frozenAccount.Status)
	require.Equal(t, user.ID, frozenAccount.FreezedBy.Int64)

	requireNoAuditLog(t, user.ID, "freeze_account", account.ID)
}

func TestFreezeAccountTxByAdmin(t *testing.T) {
	store := NewStore(testStore.db)

	ctx := context.Background()

	user := createNewUser(t)
	account := createTestAccount(t, store, user.ID, "USD", int64(3000))
	admin := CreateAdminUserTest(t)

	const (
		ipAddress = "192.168.1.10"
		userAgent = "freeze-account-test"
	)

	rowsAffected, err := testStore.FreezeAccountTx(ctx, FreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    user.ID,
		ActorID:   admin.ID,
		ActorRole: admin.Role,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	frozenAccount, err := testStore.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "frozen", frozenAccount.Status)
	require.Equal(t, admin.ID, frozenAccount.FreezedBy.Int64)

	requireAuditLog(
		t,
		admin.ID,
		"freeze_account",
		"account",
		account.ID,
		ipAddress,
		userAgent,
		map[string]any{
			"reason": "Admin-initiated account freeze",
		},
	)
}
