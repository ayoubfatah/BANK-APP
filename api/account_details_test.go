package api

import (
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
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func randomAccountDetailRow(accountID, userID int64) db.GetAccountDetailsWithUserIDRow {
	return db.GetAccountDetailsWithUserIDRow{
		ID:        5,
		AccountID: accountID,
		Type:      "iban",
		Value:     "MA64011519000001205000534921",
		Country:   pgtype.Text{String: "MA", Valid: true},
		BankName:  pgtype.Text{String: "CIH", Valid: true},
		UserID:    userID,
		// add any other fields your generated Row has
	}
}

// Matches the Tx params but ignores Actor (built by actorFromCtx).
type updateDetailParamsMatcher struct {
	want store.UpdateAccountDetailByAccountIDTxParams
}

func (m updateDetailParamsMatcher) Matches(x any) bool {
	got, ok := x.(store.UpdateAccountDetailByAccountIDTxParams)
	return ok &&
		got.AccountID == m.want.AccountID &&
		got.Type == m.want.Type &&
		got.Value == m.want.Value &&
		got.Country == m.want.Country &&
		got.BankName == m.want.BankName
}

func (m updateDetailParamsMatcher) String() string {
	return fmt.Sprintf("matches params %+v", m.want)
}

func TestGetAccountDetails(t *testing.T) {
	user := randomUser()
	other := randomUser()
	other.ID = user.ID + 1
	admin := randomAdminUser()

	const accountID int64 = 10
	url := fmt.Sprintf("/accounts/%d/details", accountID)

	userAuth := func(t *testing.T, r *http.Request, m token.Maker) {
		addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
	}
	adminAuth := func(t *testing.T, r *http.Request, m token.Maker) {
		addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
	}

	ownRow := randomAccountDetailRow(accountID, user.ID)
	otherRow := randomAccountDetailRow(accountID, other.ID)

	testCases := []TestCase{
		{
			name:      "OK_UserOwnsAccount",
			setupAuth: userAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(ownRow, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				var got struct {
					Data db.GetAccountDetailsWithUserIDRow `json:"data"`
				}
				require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
				require.Equal(t, ownRow, got.Data)
			},
		},
		{
			name:      "OK_AdminViewsAnyAccount",
			setupAuth: adminAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(otherRow, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name:      "Forbidden_UserDoesNotOwnAccount",
			setupAuth: userAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(otherRow, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusForbidden, rec.Code)
			},
		},
		{
			// Fails until the handler checks err BEFORE ownership (see bug note).
			name:      "NotFound_NonAdmin",
			setupAuth: userAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(db.GetAccountDetailsWithUserIDRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name:      "NotFound_Admin",
			setupAuth: adminAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(db.GetAccountDetailsWithUserIDRow{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name:      "InternalError_Admin",
			setupAuth: adminAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetAccountDetailsWithUserID(gomock.Any(), gomock.Eq(accountID)).
					Times(1).Return(db.GetAccountDetailsWithUserIDRow{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, url, http.MethodGet)
		})
	}
}

func TestGetAccountDetailsInvalidID(t *testing.T) {
	for _, id := range []string{"abc", "0"} {
		t.Run("id_"+id, func(t *testing.T) {
			user := randomUser()
			server := newTestServer(t, nil)
			recorder := httptest.NewRecorder()

			request, err := http.NewRequest(http.MethodGet, "/accounts/"+id+"/details", nil)
			require.NoError(t, err)
			addAuthorization(t, request, server.tokenMaker, "bearer", user.Email, user.ID, user.Role, time.Minute)

			server.router.ServeHTTP(recorder, request)
			require.Equal(t, http.StatusBadRequest, recorder.Code)
		})
	}
}

func TestUpdateAccountDetails(t *testing.T) {
	admin := randomAdminUser()

	const accountID int64 = 10

	// Change these two lines to match the route registered in server.go.
	method := http.MethodPatch
	url := fmt.Sprintf("/admin/accounts/%d/details", accountID)

	updated := db.AccountDetail{
		ID:        5,
		AccountID: accountID,
		Type:      "iban",
		Value:     "MA64",
		Country:   pgtype.Text{String: "MA", Valid: true},
		BankName:  pgtype.Text{String: "CIH", Valid: true},
	}

	adminAuth := func(t *testing.T, r *http.Request, m token.Maker) {
		addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
	}

	okStubs := func(s *mockdb.MockStore) {
		s.EXPECT().
			UpdateAccountDetailByAccountIDTx(gomock.Any(), gomock.Any()).
			Times(1).
			Return(updated, nil)
	}

	okCheck := func(t *testing.T, rec *httptest.ResponseRecorder) {
		require.Equal(t, http.StatusOK, rec.Code)
		var got db.AccountDetail
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))
		require.Equal(t, updated, got)
	}

	noCallStubs := func(s *mockdb.MockStore) {
		s.EXPECT().
			UpdateAccountDetailByAccountIDTx(gomock.Any(), gomock.Any()).
			Times(0)
	}

	testCases := []TestCase{
		{
			name:          "OK_AllFields",
			body:          map[string]any{"type": "iban", "value": "MA64", "country": "MA", "bank_name": "CIH"},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:          "OK_NoFields",
			body:          map[string]any{},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:          "OK_OnlyCountry",
			body:          map[string]any{"country": "FR"},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:          "OK_OnlyBankName",
			body:          map[string]any{"bank_name": "BMCE"},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:          "OK_OnlyType",
			body:          map[string]any{"type": "swift"},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:          "OK_OnlyValue",
			body:          map[string]any{"value": "ABC123"},
			setupAuth:     adminAuth,
			buildStubs:    okStubs,
			checkResponse: okCheck,
		},
		{
			name:      "NotFound",
			body:      map[string]any{"type": "iban"},
			setupAuth: adminAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					UpdateAccountDetailByAccountIDTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.AccountDetail{}, pgx.ErrNoRows)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name:      "InternalError",
			body:      map[string]any{"type": "iban"},
			setupAuth: adminAuth,
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					UpdateAccountDetailByAccountIDTx(gomock.Any(), gomock.Any()).
					Times(1).
					Return(db.AccountDetail{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name:       "InvalidBody_NotAnObject",
			body:       "not json", // marshals to a JSON string, can't bind into the struct
			setupAuth:  adminAuth,
			buildStubs: noCallStubs,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name:       "InvalidBody_WrongFieldType",
			body:       map[string]any{"type": 123},
			setupAuth:  adminAuth,
			buildStubs: noCallStubs,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
		{
			name:       "InvalidBody_Empty", // no body at all, EOF bind error
			body:       nil,
			setupAuth:  adminAuth,
			buildStubs: noCallStubs,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, url, method)
		})
	}
}

func TestUpdateAccountDetailsInvalidID(t *testing.T) {
	admin := randomAdminUser()

	// Same method as above.
	method := http.MethodPatch

	for _, id := range []string{"abc", "0"} {
		tc := TestCase{
			name: "id_" + id,
			body: map[string]any{"type": "iban"},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					UpdateAccountDetailByAccountIDTx(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}

		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, "/admin/accounts/"+id+"/details", method)
		})
	}
}
