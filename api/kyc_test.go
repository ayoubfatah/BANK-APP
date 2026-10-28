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
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func requireBodyMatchKYCStatus(
	t *testing.T,
	body *bytes.Buffer,
	expectedStatus string,
) {
	var got kycSummary

	err := json.Unmarshal(body.Bytes(), &got)
	require.NoError(t, err)
	require.Equal(t, expectedStatus, got.Status)
}

type TestCase struct {
	name          string
	body          any
	setupAuth     func(t *testing.T, r *http.Request, m token.Maker)
	buildStubs    func(s *mockdb.MockStore)
	checkResponse func(t *testing.T, rec *httptest.ResponseRecorder)
}

func TestGetKYC(t *testing.T) {
	user := randomUser()

	testCases := []TestCase{{
		name: "OK",
		body: nil,
		setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
			addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
		},
		buildStubs: func(s *mockdb.MockStore) {
			s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.KycVerification{}, pgx.ErrNoRows)

		},
		checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
			require.Equal(t, http.StatusOK, rec.Code)
			requireBodyMatchKYCStatus(t, rec.Body, "not_started")

		},
	},
		{
			name: "KycExistAndPending",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.KycVerification{
						Status: "pending",
					}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchKYCStatus(t, rec.Body, "pending")

			},
		},
		{
			name: "rejectedKycWithReason",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)

			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(
					db.KycVerification{
						Status: "rejected",
						RejectionReason: pgtype.Text{
							String: "Idk just got here",
							Valid:  true,
						},
					}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchKYCStatus(t, rec.Body, "rejected")

			},
		},
		{
			name: "KycExistAndApproved",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(
					t, r, m, "bearer",
					user.Email, user.ID, user.Role, time.Minute,
				)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.KycVerification{
						Status: "approved",
					}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
				requireBodyMatchKYCStatus(t, rec.Body, "approved")
			},
		},
		{
			name: "InternalServerError",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(
					t, r, m, "bearer",
					user.Email, user.ID, user.Role, time.Minute,
				)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(1).
					Return(db.KycVerification{}, errors.New("database error"))
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusInternalServerError, rec.Code)
			},
		},
		{
			name: "NotAuthorized",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).
					Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)

			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, "/users/me/kyc", http.MethodGet)
		})
	}
}

func TestGetKYCAdmin(t *testing.T) {
	user := randomAdminUser()
	kycRecords := []db.KycVerification{
		{
			ID:           1,
			UserID:       101,
			DocumentType: "passport",
			Status:       "pending",
			CreatedAt: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
		{
			ID:           2,
			UserID:       102,
			DocumentType: "national_id",
			Status:       "approved",
			CreatedAt: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
		{
			ID:           3,
			UserID:       103,
			DocumentType: "passport",
			Status:       "rejected",
			CreatedAt: pgtype.Timestamptz{
				Time:  time.Now(),
				Valid: true,
			},
		},
	}
	testCases := []TestCase{
		{

			name: "ListKYC",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},

			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					ListKYCVerifications(gomock.Any(), db.ListKYCVerificationsParams{Limit: 20, Offset: 0}).
					Times(1).
					Return(kycRecords, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {

				require.Equal(t, http.StatusOK, rec.Code)
				var got struct {
					Data   []adminKYCResponse `json:"data"`
					Limit  int32              `json:"limit"`
					Offset int32              `json:"offset"`
				}

				err := json.Unmarshal(rec.Body.Bytes(), &got)
				require.NoError(t, err)
				require.Len(t, got.Data, 3)
				require.Equal(t, int32(20), got.Limit)
				require.Equal(t, int32(0), got.Offset)
				require.Equal(t, "pending", got.Data[0].Status)
				require.Equal(t, "approved", got.Data[1].Status)
				require.Equal(t, "rejected", got.Data[2].Status)

			},
		},
		{
			name: "NotAuthorized",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {

			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					ListKYCVerifications(gomock.Any(), gomock.Any()).
					Times(0)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnauthorized, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, "/admin/kyc", http.MethodGet)
		})
	}
}

func TestSubmitKYC(t *testing.T) {
	user := randomUser()
	testCases := []TestCase{
		{
			name: "Post Kyc verification",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			body: gin.H{
				"document_type":   "passport",
				"document_number": "MA123456789",
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.KycVerification{
					Status: "not_started",
				}, nil)

				s.EXPECT().
					CreateKYCVerification(
						gomock.Any(),
						gomock.Cond(func(arg db.CreateKYCVerificationParams) bool {
							return arg.UserID == user.ID &&
								arg.DocumentType == "passport" &&
								arg.DocumentNumber.Valid &&
								arg.DocumentNumber.String != "" &&
								arg.Status == "pending"
						}),
					).
					Return(db.KycVerification{
						ID:           1,
						UserID:       user.ID,
						DocumentType: "passport",
						Status:       "pending",
					}, nil)
				s.EXPECT().CreateAuditLog(gomock.Any(), gomock.Any()).Times(1)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				t.Log("Response body:", rec.Body.String())

				require.Equal(t, http.StatusCreated, rec.Code)
				requireBodyMatchKYCStatus(t, rec.Body, "pending")
			},
		},
		{
			name: "OK_ResubmitAfterRejection",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			body: gin.H{
				"document_type":   "national_id",
				"document_number": "AB987654321",
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.KycVerification{
					Status: "rejected",
				}, nil)
				s.EXPECT().CreateKYCVerification(gomock.Any(), gomock.Any()).Return(db.KycVerification{
					ID:           2,
					UserID:       user.ID,
					DocumentType: "national_id",
					Status:       "pending",
				}, nil)
				s.EXPECT().CreateAuditLog(gomock.Any(), gomock.Any()).Times(1)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusCreated, rec.Code)
				requireBodyMatchKYCStatus(t, rec.Body, "pending")
			},
		}, {
			name: "Conflict_AlreadyPending",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			body: gin.H{
				"document_type":   "passport",
				"document_number": "MA123456789",
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.KycVerification{
					Status: "pending",
				}, nil)
				// CreateKYCVerification should NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "Conflict_AlreadyVerified",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			body: gin.H{
				"document_type":   "passport",
				"document_number": "MA123456789",
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().GetLatestKYCByUser(gomock.Any(), gomock.Eq(user.ID)).Times(1).Return(db.KycVerification{
					Status: "verified", // Matches your handler's check
				}, nil)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "BadRequest_InvalidDocumentType",
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", user.Email, user.ID, user.Role, time.Minute)
			},
			body: gin.H{
				"document_type":   "invalid_type", // Fails the "oneof" binding
				"document_number": "MA123456789",
			},
			buildStubs: func(s *mockdb.MockStore) {
				// Fails at ShouldBindJSON, so no store calls are expected
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		}}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, "/users/me/kyc", http.MethodPost)
		})
	}
}
func TestAdminRejectKYC(t *testing.T) {
	admin := randomAdminUser()
	targetKYC := db.KycVerification{
		ID:           1,
		UserID:       101,
		DocumentType: "passport",
		Status:       "pending",
	}

	validBody := gin.H{"rejection_reason": "Document is blurry and unreadable"}

	testCases := []TestCase{
		{
			name: "OK_RejectPending",
			body: validBody,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				gomock.InOrder(
					s.EXPECT().
						GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(targetKYC, nil),

					s.EXPECT().
						RejectKYC(gomock.Any(), gomock.Any()).
						Times(1).
						Return(db.KycVerification{
							ID:     targetKYC.ID,
							UserID: targetKYC.UserID,
							Status: "rejected",
							RejectionReason: pgtype.Text{
								String: "Document is blurry and unreadable",
								Valid:  true,
							},
						}, nil),

					s.EXPECT().
						CreateAuditLog(gomock.Any(), gomock.Any()).
						Times(1).
						Return(nil),
				)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "NotFound_KYCDoesNotExist",
			body: validBody,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
					Times(1).
					Return(db.KycVerification{}, pgx.ErrNoRows)
				// RejectKYC and CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "Conflict_RejectAlreadyRejected",
			body: validBody,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				rejectedKYC := targetKYC
				rejectedKYC.Status = "rejected"

				s.EXPECT().
					GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
					Times(1).
					Return(rejectedKYC, nil)
				// RejectKYC and CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "Conflict_ReviewedByAnotherAdminMidRequest",
			body: validBody,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				gomock.InOrder(
					s.EXPECT().
						GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(targetKYC, nil),

					// still pending at check time, but the UPDATE matches no rows
					s.EXPECT().
						RejectKYC(gomock.Any(), gomock.Any()).
						Times(1).
						Return(db.KycVerification{}, pgx.ErrNoRows),
				)
				// CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "BadRequest_MissingReason",
			body: gin.H{},
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				// Fails at ShouldBindJSON, so no store calls are expected
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusBadRequest, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/kyc/%d/reject", targetKYC.ID), http.MethodPatch)
		})
	}
}

func TestAdminVerifyKYC(t *testing.T) {
	admin := randomAdminUser()
	targetKYC := db.KycVerification{
		ID:           1,
		UserID:       101,
		DocumentType: "passport",
		Status:       "pending",
	}

	testCases := []TestCase{
		{
			name: "OK_VerifyPending",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				gomock.InOrder(
					s.EXPECT().
						GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(targetKYC, nil),

					s.EXPECT().
						VerifyKYC(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(db.KycVerification{
							ID:           targetKYC.ID,
							UserID:       targetKYC.UserID,
							DocumentType: targetKYC.DocumentType,
							Status:       "verified",
						}, nil),

					s.EXPECT().
						CreateAuditLog(gomock.Any(), gomock.Any()).
						Times(1).
						Return(nil),
				)
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, rec.Code)
			},
		},
		{
			name: "NotFound_KYCDoesNotExist",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				s.EXPECT().
					GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
					Times(1).
					Return(db.KycVerification{}, pgx.ErrNoRows)
				// VerifyKYC and CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusNotFound, rec.Code)
			},
		},
		{
			name: "Conflict_VerifyAlreadyVerified",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				verifiedKYC := targetKYC
				verifiedKYC.Status = "verified"

				s.EXPECT().
					GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
					Times(1).
					Return(verifiedKYC, nil)
				// VerifyKYC and CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
		{
			name: "Conflict_ReviewedByAnotherAdminMidRequest",
			body: nil,
			setupAuth: func(t *testing.T, r *http.Request, m token.Maker) {
				addAuthorization(t, r, m, "bearer", admin.Email, admin.ID, admin.Role, time.Minute)
			},
			buildStubs: func(s *mockdb.MockStore) {
				gomock.InOrder(
					s.EXPECT().
						GetKYCVerification(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(targetKYC, nil),

					// still pending at check time, but the UPDATE matches no rows
					s.EXPECT().
						VerifyKYC(gomock.Any(), gomock.Eq(targetKYC.ID)).
						Times(1).
						Return(db.KycVerification{}, pgx.ErrNoRows),
				)
				// CreateAuditLog must NOT be called
			},
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusConflict, rec.Code)
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			RunCase(t, tc, fmt.Sprintf("/admin/kyc/%d/verify", targetKYC.ID), http.MethodPatch)
		})
	}
}
