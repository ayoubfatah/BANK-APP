package db

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)


func createToken(t *testing.T) (User,RefreshToken){
    user := createNewUser(t)
	arg := CreateRefreshTokenParams{
		UserID: user.ID,
		TokenHash: utils.RandomString(10),	
		ExpiresAt: pgtype.Timestamptz{
    	Time:  time.Now().AddDate(0, 0, 30),
    	Valid: true,
	},
	}
	refreshToken, err :=	testQueries.CreateRefreshToken(context.Background(), arg)
	
	require.NoError(t, err)
	require.NotEmpty(t, refreshToken)
	require.Equal(t, refreshToken.UserID , user.ID)
	require.Equal(t, refreshToken.ExpiresAt , arg.ExpiresAt)
	require.Equal(t, refreshToken.TokenHash , arg.TokenHash)
	require.False(t, refreshToken.RevokedAt.Valid)
	return user, refreshToken
}
// CREATE TOKEN
func TestCreateRefreshToken(t *testing.T) {
 createToken(t)
}
// GET TOKEN 

func TestGetTokenByHash(t *testing.T) {
 _, token :=createToken(t)
 
 	tokenFetched , err := testQueries.GetRefreshTokenByHash(context.Background(), token.TokenHash)

 	require.NoError(t,err)
 	require.NotEmpty(t, tokenFetched)
	require.Equal(t, tokenFetched.ID , token.ID)
	require.Equal(t, tokenFetched.UserID , token.UserID)
	require.Equal(t, tokenFetched.CreatedAt , token.CreatedAt)
	require.Equal(t, tokenFetched.ExpiresAt, token.ExpiresAt)
	require.Equal(t, tokenFetched.RevokedAt, token.RevokedAt)
	require.False(t, tokenFetched.RevokedAt.Valid)
}
// ListRefreshTokensByUserID

func TestListOfRefreshTokensByUserID(t *testing.T) {
	user := createNewUser(t)


	for i := 0; i < 5; i++ {
		arg := CreateRefreshTokenParams{
			UserID:    user.ID,
			TokenHash: utils.RandomString(10),
			ExpiresAt: pgtype.Timestamptz{
				Time:  time.Now().AddDate(0, 0, 30),
				Valid: true,
			},
		}

		_, err := testQueries.CreateRefreshToken(context.Background(), arg)
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
		require.NotEmpty(t, token.TokenHash)
		require.True(t, token.ExpiresAt.Valid)
	}
}

// RevokeRefreshToken


func TestRevokeRefreshToken(t *testing.T){
	_, token :=createToken(t)
 	require.NotEmpty(t, token)
	 require.False(t, token.RevokedAt.Valid)

	revokedToken , err :=	testQueries.RevokeRefreshToken(context.Background(), token.TokenHash)
	require.NoError(t, err)


     require.WithinDuration(
		t,
		time.Now(),
		revokedToken.RevokedAt.Time,
		time.Second,
	)
	require.False(t, revokedToken.RevokedAt.Time.IsZero())
	 require.True(t, token.RevokedAt.Valid)
}



// DeleteExpiredRefreshTokens

func TestDeleteExpiredRefreshTokens(t *testing.T) {
	user := createNewUser(t)

	arg := CreateRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: utils.RandomString(10),
		ExpiresAt: pgtype.Timestamptz{
			Time:  time.Now().Add(-24 * time.Hour),
			Valid: true,
		},
	}

	token, err := testQueries.CreateRefreshToken(
		context.Background(),
		arg,
	)
	require.NoError(t, err)

	// Delete expired tokens
	err = testQueries.DeleteExpiredRefreshTokens(context.Background())
	require.NoError(t, err)

	// Token should no longer exist
	_, err = testQueries.GetRefreshTokenByHash(
		context.Background(),
		token.TokenHash,
	)
	require.ErrorIs(t, err, sql.ErrNoRows)
	

}

