package store

import (
	"context"
	"encoding/json"
	"testing"

	db "github.com/bank-app/db/sqlc"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func requireAuditLog(
	t *testing.T,
	actorID int64,
	action string,
	entityType string,
	entityID int64,
	ipAddress string,
	userAgent string,
	wantMetadata map[string]any,
) {
	t.Helper()

	ctx := context.Background()

	var (
		gotEntityType pgtype.Text
		gotEntityID   pgtype.Int8
		gotIP         pgtype.Text
		gotUserAgent  pgtype.Text
		rawMetadata   []byte
	)

	err := testDB.QueryRow(ctx, `
		SELECT entity_type, entity_id, ip_address, user_agent, metadata
		FROM audit_logs
		WHERE user_id = $1
		  AND action = $2
		  AND entity_id = $3
	`, actorID, action, entityID).Scan(
		&gotEntityType,
		&gotEntityID,
		&gotIP,
		&gotUserAgent,
		&rawMetadata,
	)
	require.NoError(t, err)

	require.Equal(t, entityType, gotEntityType.String)
	require.Equal(t, entityID, gotEntityID.Int64)
	require.Equal(t, ipAddress, gotIP.String)
	require.Equal(t, userAgent, gotUserAgent.String)

	var gotMetadata map[string]any
	err = json.Unmarshal(rawMetadata, &gotMetadata)
	require.NoError(t, err)
	require.Equal(t, wantMetadata, gotMetadata)
}

func requireNoAuditLog(
	t *testing.T,
	actorID int64,
	action string,
	entityID int64,
) {
	t.Helper()

	var count int64

	err := testDB.QueryRow(context.Background(), `
		SELECT COUNT(*)
		FROM audit_logs
		WHERE user_id = $1
		  AND action = $2
		  AND entity_id = $3
	`, actorID, action, entityID).Scan(&count)

	require.NoError(t, err)
	require.Zero(t, count)
}

func CreateAdminUserTest(t *testing.T) db.User {
	t.Helper()

	user := createNewUser(t)

	_, err := testDB.Exec(
		context.Background(),
		`UPDATE users SET role = 'admin' WHERE id = $1`,
		user.ID,
	)
	require.NoError(t, err)

	user, err = testStore.GetUser(context.Background(), user.ID)
	require.NoError(t, err)
	require.Equal(t, "admin", user.Role)

	return user
}

func TestWriteAudit(t *testing.T) {
	user := CreateAdminUserTest(t)

	testCases := []struct {
		name     string
		actor    Actor
		entry    AuditEntry
		wantIP   string
		wantMeta map[string]any
	}{
		{
			name: "ValidAuditEntry",
			actor: Actor{
				ID:        user.ID,
				Role:      user.Role,
				IPAddress: "192.168.1.10",
				UserAgent: "test-agent",
			},
			entry: AuditEntry{
				Action:     "freeze_account",
				EntityType: "account",
				EntityID:   1,
				Metadata: map[string]any{
					"reason": "suspicious_activity",
				},
			},
			wantIP: "192.168.1.10",
			wantMeta: map[string]any{
				"reason": "suspicious_activity",
			},
		},
		{
			name: "EmptyMetadataAndInvalidIP",
			actor: Actor{
				ID:        user.ID,
				Role:      user.Role,
				IPAddress: "invalid-ip",
			},
			entry: AuditEntry{
				Action: "test_action",
			},
			wantIP:   "",
			wantMeta: map[string]any{},
		},
	}

	for _, tc := range testCases {

		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			err := writeAudit(ctx, db.New(testDB), tc.actor, tc.entry)

			require.NoError(t, err)

			var (
				action     string
				entityType *string
				entityID   *int64
				ipAddress  *string
				userAgent  *string
				rawMeta    []byte
			)

			err = testDB.QueryRow(ctx, `
				SELECT action, entity_type, entity_id, ip_address,
				       user_agent, metadata
				FROM audit_logs
				WHERE user_id = $1 AND action = $2
				ORDER BY id DESC
				LIMIT 1
			`, tc.actor.ID, tc.entry.Action).Scan(
				&action,
				&entityType,
				&entityID,
				&ipAddress,
				&userAgent,
				&rawMeta,
			)
			require.NoError(t, err)

			require.Equal(t, tc.entry.Action, action)

			if tc.entry.EntityType == "" {
				require.Nil(t, entityType)
			} else {
				require.NotNil(t, entityType)
				require.Equal(t, tc.entry.EntityType, *entityType)
			}

			if tc.entry.EntityID == 0 {
				require.Nil(t, entityID)
			} else {
				require.NotNil(t, entityID)
				require.Equal(t, tc.entry.EntityID, *entityID)
			}

			if tc.wantIP == "" {
				require.Nil(t, ipAddress)
			} else {
				require.NotNil(t, ipAddress)
				require.Equal(t, tc.wantIP, *ipAddress)
			}

			if tc.actor.UserAgent == "" {
				require.Nil(t, userAgent)
			} else {
				require.NotNil(t, userAgent)
				require.Equal(t, tc.actor.UserAgent, *userAgent)
			}

			var metadata map[string]any
			err = json.Unmarshal(rawMeta, &metadata)
			require.NoError(t, err)
			require.Equal(t, tc.wantMeta, metadata)
		})
	}
}
