package api

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mock_store "github.com/bank-app/db/mock"
	mockdb "github.com/bank-app/db/mock"
	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/token"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func randomUser() db.User {
	return db.User{
		ID:        int64(rand.Intn(1000) + 1),
		FirstName: utils.RandomFirstName(),
		LastName:  randomString(8),
		Email:     utils.RandomEmail(),
		Role:      "customer",
		Status:    "active",
	}
}

func requireBodyMatchUser(t *testing.T, body *bytes.Buffer, want db.User) {
	var got userResponse
	require.NoError(t, json.Unmarshal(body.Bytes(), &got))

	require.Equal(t, want.ID, got.ID)
	require.Equal(t, want.FirstName, got.FirstName)
	require.Equal(t, want.LastName, got.LastName)
	require.Equal(t, want.Email, got.Email)
	require.Equal(t, want.Role, got.Role)
	if want.Img.Valid {
		require.NotNil(t, got.Img)
		require.Equal(t, want.Img.String, *got.Img)
	} else {
		require.Nil(t, got.Img)
	}
	if want.Phone.Valid {
		require.NotNil(t, got.Phone)
		require.Equal(t, want.Phone.String, *got.Phone)
	} else {
		require.Nil(t, got.Phone)
	}

	require.Equal(t, want.Status, got.Status)
	require.Equal(t, want.CreatedAt.Time, got.CreatedAt)
	require.Equal(t, want.UpdatedAt.Time, got.UpdatedAt)

	require.NotContains(t, body.String(), "password_hash")
	require.NotContains(t, body.String(), "password")
}

func TestGetUser(t *testing.T) {
	user := randomUser()

	testCases := []struct {
		name          string
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchUser(t, rec.Body, user)
			},
		},
		{
			name:      "NoAuthorization",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
		{
			name: "NotFound",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.User{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestUpdateUser(t *testing.T) {
	user := randomUser()
	newFirstName := "UpdatedFirst"
	newLastName := "UpdatedLast"

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"first_name": newFirstName, "last_name": newLastName},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
				updatedUser := user
				updatedUser.FirstName = newFirstName
				updatedUser.LastName = newLastName
				s.EXPECT().UpdateUser(gomock.Any(), gomock.Any()).Times(1).Return(updatedUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got userResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, newFirstName, got.FirstName)
			},
		},
		{
			name:      "NoAuthorization",
			body:      gin.H{"first_name": newFirstName},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Any()).Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPatch, "/users/me", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestUpdateUserEmail(t *testing.T) {
	user := randomUser()
	password := "secret123"
	hashedPassword, _ := utils.HashPassword(password)
	user.PasswordHash = hashedPassword
	newEmail := "new@example.com"

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"email": newEmail, "current_password": password},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
				updatedUser := user
				updatedUser.Email = newEmail
				s.EXPECT().UpdateUserEmail(gomock.Any(), gomock.Any()).Times(1).Return(updatedUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got userResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, newEmail, got.Email)
			},
		},
		{
			name: "InvalidPassword",
			body: gin.H{"email": newEmail, "current_password": "wrongpassword"},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPatch, "/users/email", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestUpdateUserPassword(t *testing.T) {
	user := randomUser()
	oldPassword := "secret123"
	hashedPassword, _ := utils.HashPassword(oldPassword)
	user.PasswordHash = hashedPassword
	newPassword := "newsecret123"

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"current_password": oldPassword, "new_password": newPassword},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
				s.EXPECT().UpdateUserPassword(gomock.Any(), gomock.Any()).Times(1).Return(nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNoContent, rec.Code)
			},
		},
		{
			name: "InvalidCurrentPassword",
			body: gin.H{"current_password": "wrongpassword", "new_password": newPassword},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPatch, "/users/password", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestUpdateUserPhone(t *testing.T) {
	user := randomUser()
	newPhone := "+1234567890"

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"phone": newPhone},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().UpdateUserPhone(gomock.Any(), gomock.Any()).Times(1).Return(nil)
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
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPatch, "/users/phone", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestCloseUserAccount(t *testing.T) {
	user := randomUser()
	password := "secret123"
	hashedPassword, _ := utils.HashPassword(password)
	user.PasswordHash = hashedPassword

	testCases := []struct {
		name          string
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{"password": password},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
				s.EXPECT().ListAccountsByUserID(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return([]db.Account{}, nil)
				closedUser := user
				closedUser.Status = "closed"
				s.EXPECT().CloseUserAccount(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(closedUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},

		{
			name: "NonZeroBalance",
			body: gin.H{"password": password},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(user, nil)
				s.EXPECT().ListAccountsByUserID(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return([]db.Account{{Balance: 100}}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()
			mockStore := mock_store.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPost, "/users/close", bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}
