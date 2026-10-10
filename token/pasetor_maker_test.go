package token

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/chacha20poly1305"
)

func TestNewPasetoMaker(t *testing.T) {
	testCases := []struct {
		name        string
		key         string
		wantErr     bool
	}{
		{
			name:    "valid key",
			key:     "12345678901234567890123456789012",
			wantErr: false,
		},
		{
			name:    "key too short",
			key:     "short",
			wantErr: true,
		},
		{
			name:    "key too long",
			key:     "123456789012345678901234567890123",
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			maker, err := NewPasetoMaker(tc.key)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, maker)
			} else {
				require.NoError(t, err)
				require.NotNil(t, maker)
			}
		})
	}
}

func TestCreateToken(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	email := "ayoub@example.com"
	duration := time.Minute

	token, _,err := maker.CreateToken(email, 1 , "customer" , duration)

	require.NoError(t, err)
	require.NotEmpty(t, token)

	payload, err := maker.VerifyToken(token)

	require.NoError(t, err)
	require.Equal(t, email, payload.Email)
	require.NotEmpty(t, payload.ID)
	require.WithinDuration(t, time.Now(), payload.IssuedAt, time.Second)
	require.True(t, payload.ExpiredAt.Equal(payload.IssuedAt.Add(duration)))
}

func TestVerifyToken(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	testCases := []struct {
		name      string
		tokenFunc func() string
		wantErr   bool
	}{
		{
			name: "valid token",
			tokenFunc: func() string {
				token,_, err := maker.CreateToken("ayoub@example.com",1 , "customer" ,time.Minute)
				require.NoError(t, err)
				return token
			},
			wantErr: false,
		},
		{
			name: "expired token",
			tokenFunc: func() string {
				token,_, err := maker.CreateToken("ayoub@example.com",1 , "customer" , -time.Minute)
				require.NoError(t, err)
				return token
			},
			wantErr: true,
		},
		{
			name: "invalid token",
			tokenFunc: func() string {
				return "invalid-token"
			},
			wantErr: true,
		},
		{
			name: "empty token",
			tokenFunc: func() string {
				return ""
			},
			wantErr: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			token := tc.tokenFunc()

			payload, err := maker.VerifyToken(token)

			if tc.wantErr {
				require.Error(t, err)
				require.Nil(t, payload)
			} else {
				require.NoError(t, err)
				require.NotNil(t, payload)
			}
		})
	}
}

func TestVerifyTokenWrongKey(t *testing.T) {
	maker1, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	maker2, err := NewPasetoMaker("abcdefghijklmnopqrstuvwxyz123456")
	require.NoError(t, err)

	token,_, err := maker1.CreateToken("ayoub@example.com", 1 , "admin" , time.Minute)
	require.NoError(t, err)

	payload, err := maker2.VerifyToken(token)

	require.Error(t, err)
	require.Nil(t, payload)
}

func TestVerifyTokenTampered(t *testing.T) {
	maker, err := NewPasetoMaker("12345678901234567890123456789012")
	require.NoError(t, err)

	token,_, err := maker.CreateToken("ayoub@example.com", 2 , "admin" , time.Minute)

	require.NoError(t, err)

	tamperedToken := token[:len(token)-1] + "x"

	payload, err := maker.VerifyToken(tamperedToken)

	require.Error(t, err)
	require.Nil(t, payload)
}

func TestPasetoMakerKeySize(t *testing.T) {
	require.Equal(t, 32, chacha20poly1305.KeySize)
}
