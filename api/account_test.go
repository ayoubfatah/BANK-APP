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
	"github.com/bank-app/db/store"
	"github.com/bank-app/token"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func requireBodyMatchAccountResult(t *testing.T,
	body *bytes.Buffer,
	expectedResult createAccountResponse) {
	var got createAccountResponse
	err := json.Unmarshal(body.Bytes(), &got)
	require.NoError(t, err)
	require.Equal(t, expectedResult, got)
}

func TestCreateAccount(t *testing.T) {
	user := randomUser()

	expectedResult := store.CreateAccountTxResult{
		Account: db.Account{
			ID:       1,
			UserID:   user.ID,
			Currency: "USD",
			Balance:  0,
			Status:   "active",
			CreatedAt: pgtype.Timestamptz{
				Time:  time.Now().UTC(),
				Valid: true,
			},
		},
		AccountDetails: db.AccountDetail{
			ID:        1,
			AccountID: 1,
			Type:      "IBAN",
			BankName: pgtype.Text{
				String: "CIH Bank",
				Valid:  true,
			},
			Country: pgtype.Text{
				String: "MA",
				Valid:  true,
			},
			Value: "MA123456789012345678901234",
		},
		BankPartner: db.BankingPartner{
			ID:           1,
			ProviderName: "CIH Bank",
			CurrencyCode: "USD",
			CountryCode:  "MA",
			DetailType:   "IBAN",
			IsActive:     true,
		},
	}

	testCases := []TestCase{
		{
			name: "OK",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(
						gomock.Any(),
						store.CreateAccountTxArguments{
							UserID:           user.ID,
							CurrencyCode:     "USD",
							BankingPartnerID: 1,
							Balance:          0,
						},
					).
					Times(1).
					Return(expectedResult, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchAccountResult(t, rec.Body, newCreateAccountResponse(expectedResult))
			},
		},
		{
			name: "CurrencyDoesntExist",
			body: CreateAccountRequest{
				CurrencyCode:     "XDX",
				BankingPartnerID: 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(
						gomock.Any(),
						gomock.Any(),
					).
					Return(store.CreateAccountTxResult{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "Binding Test",
			body: CreateAccountRequest{
				CurrencyCode:     "us",
				BankingPartnerID: 0,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(
						gomock.Any(),
						gomock.Any(),
					).Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name: "KYCDoesntExist",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(store.CreateAccountTxResult{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "KYCNotApproved",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						store.CreateAccountTxResult{},
						errors.New("User need to be verified to be able to create an account"),
					)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name: "BankingPartnerDoesntExist",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 999,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(store.CreateAccountTxResult{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "WrongBankingPartnerCurrency",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						store.CreateAccountTxResult{},
						errors.New("wrong currency"),
					)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name: "BankingPartnerInactive",
			body: gin.H{
				"currency_code":      "USD",
				"banking_partner_id": 1,
			},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					CreateAccountTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(
						store.CreateAccountTxResult{},
						errors.New("banking partner is inactive"),
					)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, "/accounts", http.MethodPost)
		})
	}
}

func TestGetAccounts(t *testing.T) {
	user := randomUser()
	accounts := []db.Account{
		{ID: 1, UserID: user.ID, Currency: "USD", Balance: 1000, Status: "active"},
		{ID: 2, UserID: user.ID, Currency: "EUR", Balance: 500, Status: "active"},
	}

	testCases := []TestCase{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().ListAccountsByUserID(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(accounts, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got []db.Account
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Len(t, got, 2)
			},
		},
		{
			name: "InternalError",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					ListAccountsByUserID(
						gomock.Any(),
						user.ID,
					).
					Times(1).
					Return(nil, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name: "InternalError",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					ListAccountsByUserID(gomock.Any(), user.ID).
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
			RunCase(t, tc, "/accounts", http.MethodGet)
		})
	}
}
func TestGetAccount(t *testing.T) {
	user := randomUser()
	account := db.Account{
		ID: 2, UserID: user.ID, Currency: "EUR", Balance: 500, Status: "active",
	}

	testCases := []TestCase{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountByIDAndUserID(gomock.Any(), gomock.Eq(db.GetAccountByIDAndUserIDParams{
					ID:     account.ID,
					UserID: user.ID,
				})).Times(1).Return(account, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got db.Account
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

			},
		},
		{
			name: "Internal server error",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccountByIDAndUserID(
						gomock.Any(),
						gomock.Eq(db.GetAccountByIDAndUserIDParams{
							ID:     account.ID,
							UserID: user.ID,
						}),
					).
					Times(1).
					Return(db.Account{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/accounts/%d", account.ID), http.MethodGet)
		})
	}
}

func TestGetAccountInvalidID(t *testing.T) {
	user := randomUser()

	server := newTestServer(t, nil) // Use your existing server setup.
	recorder := httptest.NewRecorder()

	url := "/accounts/abc"
	request, err := http.NewRequest(http.MethodGet, url, nil)
	require.NoError(t, err)

	addAuthorization(t, request, server.tokenMaker, "bearer",
		user.Email, user.ID, user.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestFreezeAccount(t *testing.T) {
	user := randomUser()
	otherUser := randomUser()
	account := db.Account{ID: 1, UserID: user.ID, Status: "active"}
	// OtherAccount := db.Account{ID: 2, UserID: otherUser.ID, Status: "active"}

	testCases := []TestCase{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
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
			name: "Account Doesn't Exist",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
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
			name: "AlreadyFrozen",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
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
			name: "Forbidden - Account Belongs To Another User",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer",
					otherUser.Email, otherUser.ID, otherUser.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(account, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			name: "Account Doesn't Exist",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer",
					user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(account.ID)).
					Times(1).
					Return(db.Account{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var got map[string]string

				err := json.Unmarshal(rec.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, http.StatusNotFound, rec.Code)
				require.Equal(t, "not found", got["error"])

			},
		},
		{
			name: "FreezeAccount db error",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(account.ID)).Times(1).Return(account, nil)
				s.EXPECT().FreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var got map[string]string
				err := json.Unmarshal(rec.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Equal(t, http.StatusNotFound, rec.Code)
				require.Equal(t, "not found", got["error"])
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/accounts/%d/freeze", account.ID), http.MethodPost)
		})
	}
}

func TestFreezeAccountInvalidAccount(t *testing.T) {
	user := randomUser()

	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/accounts/abc/freeze"
	request, err := http.NewRequest(http.MethodPost, url, nil)
	require.NoError(t, err)

	addAuthorization(t, request, server.tokenMaker, "bearer",
		user.Email, user.ID, user.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestUnFreezeAccount(t *testing.T) {
	user := randomUser()
	otherUser := randomUser()
	admin := randomAdminUser()

	// Account frozen by the current user
	accountSelfFrozen := db.Account{
		ID:        1,
		UserID:    user.ID,
		Status:    "frozen",
		FreezedBy: pgtype.Int8{Int64: user.ID, Valid: true},
	}

	// Account frozen by someone else
	accountOtherFrozen := db.Account{
		ID:        2,
		UserID:    otherUser.ID,
		Status:    "frozen",
		FreezedBy: pgtype.Int8{Int64: otherUser.ID, Valid: true},
	}

	testCases := []TestCase{
		{
			name: "OK_UnfreezeSelf",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(accountSelfFrozen.ID)).Times(1).Return(accountSelfFrozen, nil)
				s.EXPECT().UnFreezeAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(1), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "Forbidden_UnfreezeOtherUser",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(accountOtherFrozen.ID)).Times(1).Return(accountOtherFrozen, nil)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				var got map[string]string
				err := json.Unmarshal(rec.Body.Bytes(), &got)

				require.NoError(t, err)
				require.Equal(t, got["error"], "You cannot unfreeze an account that doesn't belong to you")
				require.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			name: "Account_Already_unFrozen",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer",
					admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accountOtherFrozen.ID)).
					Times(1).
					Return(accountOtherFrozen, nil)

				s.EXPECT().
					UnFreezeAccountTx(gomock.Any(), gomock.Eq(
						store.UnFreezeAccountTxParams{
							AccountID: accountOtherFrozen.ID,
							UserID:    accountOtherFrozen.UserID,
							ActorID:   admin.ID,
							ActorRole: admin.Role,
							IPAddress: "192.0.2.1",
							UserAgent: "",
						},
					)).
					Times(1).
					Return(int64(0), nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "DbError_UnFreezeAccountTx",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer",
					admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accountOtherFrozen.ID)).
					Times(1).
					Return(accountOtherFrozen, nil)

				s.EXPECT().
					UnFreezeAccountTx(gomock.Any(), gomock.Eq(
						store.UnFreezeAccountTxParams{
							AccountID: accountOtherFrozen.ID,
							UserID:    accountOtherFrozen.UserID,
							ActorID:   admin.ID,
							ActorRole: admin.Role,
							IPAddress: "192.0.2.1",
							UserAgent: "",
						},
					)).
					Times(1).
					Return(int64(0), pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "DbError_UnFreezeAccountTx",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer",
					admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetAccount(gomock.Any(), gomock.Eq(accountOtherFrozen.ID)).
					Times(1).
					Return(db.Account{}, pgx.ErrNoRows)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			if tc.name == "OK_UnfreezeSelf" {
				RunCase(t, tc, fmt.Sprintf("/accounts/%d/unfreeze", accountSelfFrozen.ID), http.MethodPost)
			} else {
				RunCase(t, tc, fmt.Sprintf("/accounts/%d/unfreeze", accountOtherFrozen.ID), http.MethodPost)
			}
		})
	}
}
func TestUnFreezeAccountFrozenByAdmin(t *testing.T) {
	admin := randomAdminUser()
	user := randomUser()

	accountAdminFrozen := db.Account{
		ID:        3,
		UserID:    user.ID,
		Status:    "frozen",
		FreezedBy: pgtype.Int8{Int64: admin.ID, Valid: true},
	}

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	st := mockdb.NewMockStore(ctrl)

	st.EXPECT().
		GetAccount(gomock.Any(), accountAdminFrozen.ID).
		Times(1).
		Return(accountAdminFrozen, nil)

	server := newTestServer(t, st)

	url := fmt.Sprintf("/accounts/%v/unfreeze", accountAdminFrozen.ID)
	request, err := http.NewRequest(http.MethodPost, url, nil)
	require.NoError(t, err)

	addAuthorization(
		t,
		request,
		server.tokenMaker,
		"bearer",
		user.Email,
		user.ID,
		user.Role,
		time.Minute,
	)

	recorder := httptest.NewRecorder()
	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusForbidden, recorder.Code)
}
func TestUnFreezeAccountInvalidAccount(t *testing.T) {
	user := randomUser()

	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/accounts/abc/unfreeze"
	request, err := http.NewRequest(http.MethodPost, url, nil)
	require.NoError(t, err)

	addAuthorization(t, request, server.tokenMaker, "bearer",
		user.Email, user.ID, user.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCloseAccountInvalidId(t *testing.T) {
	user := randomUser()

	server := newTestServer(t, nil)
	recorder := httptest.NewRecorder()

	url := "/accounts/abc/close"
	request, err := http.NewRequest(http.MethodPost, url, nil)
	require.NoError(t, err)

	addAuthorization(t, request, server.tokenMaker, "bearer",
		user.Email, user.ID, user.Role, time.Minute)

	server.router.ServeHTTP(recorder, request)

	require.Equal(t, http.StatusBadRequest, recorder.Code)
}

func TestCloseAccount(t *testing.T) {
	user := randomUser()
	otherUser := randomUser()
	activeAccount := db.Account{ID: 1, UserID: user.ID, Balance: 0, Status: "active"}
	balanceAccount := db.Account{ID: 2, UserID: user.ID, Balance: 100, Status: "active"}

	testCases := []TestCase{
		{
			name: "OK",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
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
			name: "Conflict_NonZeroBalance",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(balanceAccount.ID)).Times(1).Return(balanceAccount, nil)
				// CloseAccountTx should NOT be called because balance != 0
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "CloseOther_Account",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", otherUser.Email, otherUser.ID, otherUser.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(balanceAccount, nil)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			name: "Db_Error_GetAccount",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(db.Account{}, pgx.ErrNoRows)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "Db_Error_CloseAccount",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(activeAccount, nil)
				s.EXPECT().CloseAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), pgx.ErrNoRows)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "RowsAffected0_CloseAccount",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccount(gomock.Any(), gomock.Eq(activeAccount.ID)).Times(1).Return(activeAccount, nil)
				s.EXPECT().CloseAccountTx(gomock.Any(), gomock.Any()).Times(1).Return(int64(0), nil)

			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := activeAccount.ID
			if tc.name == "Conflict_NonZeroBalance" {
				id = balanceAccount.ID
			}
			RunCase(t, tc, fmt.Sprintf("/accounts/%d/close", id), http.MethodPost)
		})
	}
}
