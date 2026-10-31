package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	mockdb "github.com/bank-app/db/mock"
	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/token"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestAdminListAccounts(t *testing.T) {
	admin := randomAdminUser()

	// Sample accounts to return from the mock store
	mockAccounts := []db.Account{
		{ID: 1, UserID: 101, Currency: "USD", Balance: 1000, Status: "active"},
		{ID: 2, UserID: 102, Currency: "EUR", Balance: 500, Status: "frozen"},
	}

	testCases := []struct {
		name          string
		query         string
		setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
		buildStubs    func(s *mockdb.MockStore)
		checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:  "OK_DefaultPagination",
			query: "", // No query params, should default to page_id=1, page_size=20
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Eq(db.ListAccountsAdminParams{
					Limit:  20,
					Offset: 0,
				})).Times(1).Return(mockAccounts, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got map[string]interface{}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, float64(1), got["page_id"])
				require.Equal(t, float64(20), got["page_size"])
				require.Len(t, got["accounts"].([]interface{}), 2)
			},
		},
		{
			name:  "OK_WithUserIDFilter",
			query: "?user_id=101",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Eq(db.ListAccountsAdminParams{
					Limit:  20,
					Offset: 0,
					UserID: pgtype.Int8{Int64: 101, Valid: true},
				})).Times(1).Return(mockAccounts[:1], nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:  "OK_WithCurrencyFilter",
			query: "?currency=EUR",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Eq(db.ListAccountsAdminParams{
					Limit:    20,
					Offset:   0,
					Currency: pgtype.Text{String: "EUR", Valid: true},
				})).Times(1).Return(mockAccounts[1:], nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:  "OK_WithStatusFilter",
			query: "?status=frozen",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Eq(db.ListAccountsAdminParams{
					Limit:  20,
					Offset: 0,
					Status: pgtype.Text{String: "frozen", Valid: true},
				})).Times(1).Return(mockAccounts[1:], nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:  "OK_WithAllFiltersAndCustomPagination",
			query: "?page_id=2&page_size=10&user_id=101&currency=USD&status=active",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Eq(db.ListAccountsAdminParams{
					Limit:    10,
					Offset:   10, // (2 - 1) * 10
					UserID:   pgtype.Int8{Int64: 101, Valid: true},
					Currency: pgtype.Text{String: "USD", Valid: true},
					Status:   pgtype.Text{String: "active", Valid: true},
				})).Times(1).Return(mockAccounts, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got map[string]interface{}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, float64(2), got["page_id"])
				require.Equal(t, float64(10), got["page_size"])
			},
		},
		{
			name:  "InvalidQueryParams",
			query: "?page_id=abc", // Invalid integer for page_id
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// No stubs expected because query binding fails before DB call
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name:  "InternalError",
			query: "?status=active",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsAdmin(gomock.Any(), gomock.Any()).
					Times(1).
					Return(nil, errors.New("database connection failed"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			store := mockdb.NewMockStore(ctrl)
			tc.buildStubs(store)

			server := newTestServer(t, store)
			recorder := httptest.NewRecorder()

			url := fmt.Sprintf("/admin/accounts%s", tc.query)
			request, err := http.NewRequest(http.MethodGet, url, nil)
			require.NoError(t, err)

			tc.setupAuth(t, request, server.tokenMaker)

			server.router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}

func TestAdminGetAccount(t *testing.T) {
	user := randomUser()
	account := db.Account{
		ID: 1, UserID: user.ID, Currency: "USD", Balance: 1000, Status: "active",
	}

	testCases := []TestCase{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, "admin", time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got db.Account
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, account, got)
			},
		},
		{
			name: "NotFound",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, "admin", time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, "admin", time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/accounts/%d", account.ID), http.MethodGet)
		})
	}
}

func TestAdminGetAccountInvalidID(t *testing.T) {
	user := randomUser()
	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/admin/accounts/abc"
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	addAuthorization(t, request, server.tokenMaker, "bearer", user.Email, user.ID, "admin", time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAdminFreezeAccount(t *testing.T) {
	user := randomUser()
	admin := randomAdminUser()
	account := db.Account{ID: 1, UserID: user.ID, Status: "active"}

	testCases := []TestCase{
		{
			name: "OK",
			body: adminActionRequest{
				Reason: "Suspicious activity detected",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().FreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "InvalidBody",
			body: adminActionRequest{
				Reason: "x", // too short, violates min=5
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// No stubs expected because JSON binding fails
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "AccountDoesntExist",
			body: adminActionRequest{
				Reason: "Suspicious activity detected",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "AlreadyFrozen",
			body: adminActionRequest{
				Reason: "Suspicious activity detected",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().FreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "InternalError",
			body: adminActionRequest{
				Reason: "Suspicious activity detected",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().FreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/accounts/%d/freeze", account.ID), http.MethodPost)
		})
	}
}

func TestAdminFreezeAccountInvalidID(t *testing.T) {
	admin := randomAdminUser()
	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/admin/accounts/abc/freeze"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"reason": "suspicious activity"}`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	addAuthorization(t, request, server.tokenMaker, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAdminUnfreezeAccount(t *testing.T) {
	user := randomUser()
	admin := randomAdminUser()
	account := db.Account{
		ID:        1,
		UserID:    user.ID,
		Status:    "frozen",
		FreezedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}

	testCases := []TestCase{
		{
			name: "OK",
			body: adminActionRequest{
				Reason: "Investigation completed, account cleared",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().UnFreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "InvalidBody",
			body: adminActionRequest{
				Reason: "x", // too short, violates min=5
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// No stubs expected because JSON binding fails
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "AccountDoesntExist",
			body: adminActionRequest{
				Reason: "Investigation completed, account cleared",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(db.Account{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "AlreadyUnfrozen",
			body: adminActionRequest{
				Reason: "Investigation completed, account cleared",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().UnFreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "InternalError",
			body: adminActionRequest{
				Reason: "Investigation completed, account cleared",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().UnFreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/accounts/%d/unfreeze", account.ID), http.MethodPost)
		})
	}
}

func TestAdminUnfreezeAccountInvalidID(t *testing.T) {
	admin := randomAdminUser()
	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/admin/accounts/abc/unfreeze"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"reason": "cleared"}`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	addAuthorization(t, request, server.tokenMaker, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestAdminCloseAccount(t *testing.T) {
	user := randomUser()
	admin := randomAdminUser()
	activeAccount := db.Account{ID: 1, UserID: user.ID, Balance: 0, Status: "active"}

	testCases := []TestCase{
		{
			name: "OK",
			body: adminActionRequest{
				Reason: "Account closure requested by user",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(activeAccount, nil)
				s.EXPECT().CloseAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "InvalidBody",
			body: adminActionRequest{
				Reason: "x", // too short, violates min=5
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// No stubs expected because JSON binding fails
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "AccountDoesntExist",
			body: adminActionRequest{
				Reason: "Account closure requested by user",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(db.Account{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "AlreadyClosed",
			body: adminActionRequest{
				Reason: "Account closure requested by user",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(activeAccount, nil)
				s.EXPECT().CloseAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "InternalError",
			body: adminActionRequest{
				Reason: "Account closure requested by user",
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(activeAccount, nil)
				s.EXPECT().CloseAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/accounts/%d/close", activeAccount.ID), http.MethodPost)
		})
	}
}

func TestAdminCloseAccountInvalidID(t *testing.T) {
	admin := randomAdminUser()
	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/admin/accounts/abc/close"
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewBufferString(`{"reason": "closed"}`))
	require.NoError(t, err)
	request.Header.Set("Content-Type", "application/json")

	addAuthorization(t, request, server.tokenMaker, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}
