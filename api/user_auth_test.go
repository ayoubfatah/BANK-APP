package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/bank-app/db/mock"
	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/token"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateUser(t *testing.T) {
	user := randomUser()
	password := "secret123"

	testCases := []struct {
		name          string
		body          interface{}
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
				"password":   password,
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().CreateUser(gomock.Any(), gomock.Any()).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, rec.Code)
				requireBodyMatchUser(t, rec.Body, user)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      "invalid-email",
				"password":   password,
			},
			buildStubs: func(s *mockdb.MockStore) {},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "ShortPassword",
			body: gin.H{
				"first_name": user.FirstName,
				"last_name":  user.LastName,
				"email":      user.Email,
				"password":   "123",
			},
			buildStubs: func(s *mockdb.MockStore) {},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestLoginUser(t *testing.T) {
	user := randomUser()
	password := "secret123"
	hashedPassword, _ := utils.HashPassword(password)
	user.PasswordHash = hashedPassword

	testCases := []struct {
		name          string
		body          interface{}
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"email":    user.Email,
				"password": password,
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).Times(1).Return(user, nil)
				s.EXPECT().CreateRefreshToken(gomock.Any(), gomock.Any()).Times(1).Return(db.RefreshToken{}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got loginUserResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.NotEmpty(t, got.AccessToken)
				require.NotEmpty(t, got.RefreshToken)
				require.Equal(t, user.ID, got.User.ID)
			},
		},
		{
			name: "InvalidEmail",
			body: gin.H{
				"email":    "wrong@example.com",
				"password": password,
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUserByEmail(gomock.Any(), gomock.Any()).Times(1).Return(db.User{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "InvalidPassword",
			body: gin.H{
				"email":    user.Email,
				"password": "wrongpassword",
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "InactiveAccount",
			body: gin.H{
				"email":    user.Email,
				"password": password,
			},
			buildStubs: func(s *mockdb.MockStore) {
				inactiveUser := user
				inactiveUser.Status = "suspended"
				s.EXPECT().GetUserByEmail(gomock.Any(), gomock.Eq(user.Email)).Times(1).Return(inactiveUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPost, "/users/login", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestLogoutUser(t *testing.T) {
	user := randomUser()
	refreshToken := "test-refresh-token"

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"refresh_token": refreshToken,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				session := db.RefreshToken{
					UserID: user.ID,
				}
				s.EXPECT().GetRefreshTokenByHash(gomock.Any(), gomock.Eq(refreshToken)).Times(1).Return(session, nil)
				s.EXPECT().RevokeRefreshToken(gomock.Any(), gomock.Eq(refreshToken)).Times(1).Return(db.RefreshToken{}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "NoAuthorization",
			body: gin.H{
				"refresh_token": refreshToken,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetRefreshTokenByHash(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "InvalidSessionUser",
			body: gin.H{
				"refresh_token": refreshToken,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				session := db.RefreshToken{
					UserID: user.ID + 1, // Different user
				}
				s.EXPECT().GetRefreshTokenByHash(gomock.Any(), gomock.Eq(refreshToken)).Times(1).Return(session, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "AlreadyRevoked",
			body: gin.H{
				"refresh_token": refreshToken,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				session := db.RefreshToken{
					UserID:    user.ID,
					RevokedAt: pgtype.Timestamptz{Valid: true},
				}
				s.EXPECT().GetRefreshTokenByHash(gomock.Any(), gomock.Eq(refreshToken)).Times(1).Return(session, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPost, "/users/logout", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}
