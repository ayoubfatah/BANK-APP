package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
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
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func randomAdminUser() db.User {
	return db.User{
		ID:        int64(rand.Intn(1000) + 1),
		FirstName: utils.RandomFirstName(),
		LastName:  randomString(8),
		Email:     utils.RandomEmail(),
		Role:      "admin",
		Status:    "active",
	}
}

func TestListUsers(t *testing.T) {
	admin := randomAdminUser()
	users := []db.User{randomUser(), randomUser()}

	testCases := []struct {
		name          string
		query         string
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:  "OK",
			query: "?page=1&limit=10",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAllUsers(gomock.Any(), gomock.Any()).Times(1).Return(users, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:      "NoAuthorization",
			query:     "?page=1&limit=10",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAllUsers(gomock.Any(), gomock.Any()).Times(0)
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
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			request := httptest.NewRequest(http.MethodGet, "/admin/users"+tc.query, nil)
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestGetUserByAdmin(t *testing.T) {
	admin := randomAdminUser()
	targetUser := randomUser()

	testCases := []struct {
		name          string
		userID        int64
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: targetUser.ID,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(targetUser.ID)).Times(1).Return(targetUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchUser(t, rec.Body, targetUser)
			},
		},
		{
			name:   "NotFound",
			userID: 999,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetUser(gomock.Any(), gomock.Eq(int64(999))).Times(1).Return(db.User{}, pgx.ErrNoRows)
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
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			request := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/admin/users/%d", tc.userID), nil)
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestSetUserStatus(t *testing.T) {
	admin := randomAdminUser()
	targetUser := randomUser()

	testCases := []struct {
		name          string
		userID        int64
		body          interface{}
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: targetUser.ID,
			body:   gin.H{"status": "suspended"},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// requireZeroBalances is ONLY called if status == "closed"
				updatedUser := targetUser
				updatedUser.Status = "suspended"
				s.EXPECT().UpdateUserStatus(gomock.Any(), gomock.Any()).Times(1).Return(updatedUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got userResponse
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, "suspended", got.Status)
			},
		},
		{
			name:   "CannotChangeOwnStatus",
			userID: admin.ID,
			body:   gin.H{"status": "suspended"},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// No store calls expected
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name:   "NonZeroBalanceOnClose",
			userID: targetUser.ID,
			body:   gin.H{"status": "closed"},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsByUserID(gomock.Any(), gomock.Eq(targetUser.ID)).Times(1).Return([]db.Account{{Balance: 50}}, nil)
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
			mockStore := mockdb.NewMockStore(ctrl)
			tc.buildStubs(mockStore)

			server := newTestServer(t, mockStore)

			body, _ := json.Marshal(tc.body)
			request := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d/status", tc.userID), bytes.NewReader(body))
			request.Header.Set("Content-Type", "application/json")
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}

func TestCloseUserAccountByAdmin(t *testing.T) {
	admin := randomAdminUser()
	targetUser := randomUser()

	testCases := []struct {
		name          string
		userID        int64
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "OK",
			userID: targetUser.ID,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsByUserID(gomock.Any(), gomock.Eq(targetUser.ID)).Times(1).Return([]db.Account{}, nil)
				closedUser := targetUser
				closedUser.Status = "closed"
				s.EXPECT().CloseUserAccount(gomock.Any(), gomock.Eq(targetUser.ID)).Times(1).Return(closedUser, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:   "CannotCloseOwnAccount",
			userID: admin.ID,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
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

			request := httptest.NewRequest(http.MethodPatch, fmt.Sprintf("/admin/users/%d", tc.userID), nil)
			tc.setupAuth(t, request, server.tokenMaker)

			rec := httptest.NewRecorder()
			server.router.ServeHTTP(rec, request)
			tc.checkResponse(t, rec)
		})
	}
}
