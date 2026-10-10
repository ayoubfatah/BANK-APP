package db

import (
	"context"
	"database/sql"
	"net/netip"
	"testing"
	"time"

	"github.com/bank-app/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func createToken(t *testing.T) (User, RefreshToken) {
	t.Helper()

	user := createNewUser(t)

	arg := CreateRefreshTokenParams{
		ID: pgtype.UUID{
			Bytes: [16]byte(uuid.New()),
			Valid: true,
		},
		UserID:    user.ID,
		ClientIp:  netip.MustParseAddr("192.0.2.10"),
		TokenHash: utils.RandomString(32),
		UserAgent: "Go test client",
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().AddDate(0, 0, 30).Truncate(time.Microsecond),
			Valid: true,
		},
	}

	refreshToken, err := testQueries.CreateRefreshToken(
		context.Background(),
		arg,
	)

	require.NoError(t, err)
	require.Equal(t, user.ID, refreshToken.UserID)
	require.Equal(t, arg.ID, refreshToken.ID)
	require.Equal(t, arg.ClientIp, refreshToken.ClientIp)
	require.Equal(t, arg.TokenHash, refreshToken.TokenHash)
	require.Equal(t, arg.UserAgent, refreshToken.UserAgent)
	require.Equal(t, arg.ExpiresAt, refreshToken.ExpiresAt)
	require.False(t, refreshToken.IsBlocked)
	require.False(t, refreshToken.RevokedAt.Valid)

	return user, refreshToken
}

// CREATE TOKEN

func TestCreateRefreshToken(t *testing.T) {
	createToken(t)
}

// GET TOKEN BY HASH

func TestGetTokenByHash(t *testing.T) {
	_, token := createToken(t)

	tokenFetched, err := testQueries.GetRefreshTokenByHash(
		context.Background(),
		token.TokenHash,
	)

	require.NoError(t, err)
	require.Equal(t, token.ID, tokenFetched.ID)
	require.Equal(t, token.UserID, tokenFetched.UserID)
	require.Equal(t, token.ClientIp, tokenFetched.ClientIp)
	require.Equal(t, token.TokenHash, tokenFetched.TokenHash)
	require.Equal(t, token.UserAgent, tokenFetched.UserAgent)
	require.Equal(t, token.CreatedAt, tokenFetched.CreatedAt)
	require.Equal(t, token.ExpiresAt, tokenFetched.ExpiresAt)
	require.Equal(t, token.RevokedAt, tokenFetched.RevokedAt)
	require.Equal(t, token.IsBlocked, tokenFetched.IsBlocked)
	require.False(t, tokenFetched.RevokedAt.Valid)
}

// LIST REFRESH TOKENS BY USER ID

func TestListOfRefreshTokensByUserID(t *testing.T) {
	user := createNewUser(t)

	for i := 0; i < 5; i++ {
		arg := CreateRefreshTokenParams{
			ID: pgtype.UUID{
				Bytes: [16]byte(uuid.New()),
				Valid: true,
			},
			UserID:    user.ID,
			ClientIp:  netip.MustParseAddr("192.0.2.10"),
			TokenHash: utils.RandomString(32),
			UserAgent: "Go test client",
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().AddDate(0, 0, 30).Truncate(time.Microsecond),
				Valid: true,
			},
		}

		_, err := testQueries.CreateRefreshToken(
			context.Background(),
			arg,
		)
		require.NoError(t, err)
	}

	refreshTokens, err := testQueries.ListRefreshTokensByUserID(
		context.Background(),
		user.ID,
	)

	require.NoError(t, err)
	require.Len(t, refreshTokens, 5)

	for _, token := range refreshTokens {
		require.Equal(t, user.ID, token.UserID)
		require.NotEmpty(t, token.ID)
		require.NotEmpty(t, token.TokenHash)
		require.NotEmpty(t, token.UserAgent)
		require.Equal(t, netip.MustParseAddr("192.0.2.10"), token.ClientIp)
		require.True(t, token.ExpiresAt.Valid)
		require.False(t, token.IsBlocked)
		require.False(t, token.RevokedAt.Valid)
	}
}

// REVOKE REFRESH TOKEN

func TestRevokeRefreshToken(t *testing.T) {
	_, token := createToken(t)

	require.False(t, token.RevokedAt.Valid)

	revokedToken, err := testQueries.RevokeRefreshToken(
		context.Background(),
		token.TokenHash,
	)

	require.NoError(t, err)
	require.True(t, revokedToken.RevokedAt.Valid)
	require.WithinDuration(
		t,
		time.Now(),
		revokedToken.RevokedAt.Time,
		time.Second,
	)
}

// BLOCK REFRESH TOKEN

func TestBlockRefreshToken(t *testing.T) {
	_, token := createToken(t)

	require.False(t, token.IsBlocked)

	blockedToken, err := testQueries.BlockRefreshToken(
		context.Background(),
		BlockRefreshTokenParams{
			ID:        token.ID,
			IsBlocked: true,
		},
	)

	require.NoError(t, err)
	require.True(t, blockedToken.IsBlocked)
	require.Equal(t, token.ID, blockedToken.ID)
	require.Equal(t, token.UserID, blockedToken.UserID)

	unblockedToken, err := testQueries.BlockRefreshToken(
		context.Background(),
		BlockRefreshTokenParams{
			ID:        token.ID,
			IsBlocked: false,
		},
	)

	require.NoError(t, err)
	require.False(t, unblockedToken.IsBlocked)
}

// DELETE EXPIRED REFRESH TOKENS

func TestDeleteExpiredRefreshTokens(t *testing.T) {
	user := createNewUser(t)

	arg := CreateRefreshTokenParams{
		ID: pgtype.UUID{
			Bytes: [16]byte(uuid.New()),
			Valid: true,
		},
		UserID:    user.ID,
		ClientIp:  netip.MustParseAddr("192.0.2.10"),
		TokenHash: utils.RandomString(32),
		UserAgent: "Go test client",
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(-24 * time.Hour).Truncate(time.Microsecond),
			Valid: true,
		},
	}

	token, err := testQueries.CreateRefreshToken(
		context.Background(),
		arg,
	)
	require.NoError(t, err)

	err = testQueries.DeleteExpiredRefreshTokens(context.Background())
	require.NoError(t, err)

	_, err = testQueries.GetRefreshTokenByHash(
		context.Background(),
		token.TokenHash,
	)

	require.ErrorIs(t, err, sql.ErrNoRows)
}
