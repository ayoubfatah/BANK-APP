package store

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnFreezeAccountTxByUser(t *testing.T) {
	store := testStore
	ctx := context.Background()

	user := createNewUser(t)
	account := createTestAccount(t, store, user.ID, "USD", int64(3000))

	// Freeze the account as its owner.
	rowsAffected, err := store.FreezeAccountTx(ctx, FreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    user.ID,
		ActorID:   user.ID,
		ActorRole: user.Role,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	// Unfreeze the account as the same user.
	rowsAffected, err = store.UnFreezeAccountTx(ctx, UnFreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    user.ID,
		ActorID:   user.ID,
		ActorRole: user.Role,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	// Verify the account is active again.
	unfrozenAccount, err := store.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "active", unfrozenAccount.Status)

	// A user unfreezing their own account should not create an audit log.
	requireNoAuditLog(t, user.ID, "unfreeze_account", account.ID)
}

func TestUnFreezeAccountTxByDifferentUser(t *testing.T) {
	store := testStore
	ctx := context.Background()

	owner := createNewUser(t)
	otherUser := createNewUser(t)

	account := createTestAccount(t, store, owner.ID, "USD", int64(3000))

	// The owner freezes their account.
	rowsAffected, err := store.FreezeAccountTx(ctx, FreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    owner.ID,
		ActorID:   owner.ID,
		ActorRole: owner.Role,
	})
	require.NoError(t, err)
	require.Equal(t, int64(1), rowsAffected)

	// Another user attempts to unfreeze it.
	rowsAffected, err = store.UnFreezeAccountTx(ctx, UnFreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    otherUser.ID,
		ActorID:   otherUser.ID,
		ActorRole: otherUser.Role,
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected)

	// The account must remain frozen.
	frozenAccount, err := store.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "frozen", frozenAccount.Status)

	requireNoAuditLog(t, otherUser.ID, "unfreeze_account", account.ID)
}

func TestUnFreezeAccountTxByAdminWhenAlreadyUnfrozen(t *testing.T) {
	store := testStore
	ctx := context.Background()

	user := createNewUser(t)
	admin := CreateAdminUserTest(t)

	account := createTestAccount(t, store, user.ID, "USD", int64(3000))

	// The account is already active.
	rowsAffected, err := store.UnFreezeAccountTx(ctx, UnFreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    user.ID,
		ActorID:   admin.ID,
		ActorRole: admin.Role,
		IPAddress: "192.168.1.10",
		UserAgent: "unfreeze-account-test",
	})
	require.NoError(t, err)
	require.Zero(t, rowsAffected)

	// The account must remain active.
	activeAccount, err := store.GetAccount(ctx, account.ID)
	require.NoError(t, err)
	require.Equal(t, "active", activeAccount.Status)

	// No audit entry should be created when no state change occurs.
	requireNoAuditLog(t, admin.ID, "unfreeze_account", account.ID)
}
