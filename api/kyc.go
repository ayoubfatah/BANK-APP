package api

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type kycSummary struct {
	Status          string     `json:"status"`
	DocumentType    *string    `json:"document_type"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedAt      *time.Time `json:"verified_at"`
}

type adminKYCResponse struct {
	ID              int64      `json:"id"`
	UserID          int64      `json:"user_id"`
	DocumentType    string     `json:"document_type"`
	Status          string     `json:"status"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedAt      *time.Time `json:"verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

func newAdminKYCResponse(k db.KycVerification) adminKYCResponse {
	return adminKYCResponse{
		ID:              k.ID,
		UserID:          k.UserID,
		DocumentType:    k.DocumentType,
		Status:          k.Status,
		RejectionReason: ptrText(k.RejectionReason),
		VerifiedAt:      ptrTime(k.VerifiedAt),
		CreatedAt:       k.CreatedAt.Time,
	}
}

func newKYCSummary(k db.KycVerification) kycSummary {
	return kycSummary{
		Status:          k.Status,
		DocumentType:    &k.DocumentType,
		RejectionReason: ptrText(k.RejectionReason),
		VerifiedAt:      ptrTime(k.VerifiedAt),
	}
}

type submitKYCRequest struct {
	DocumentType   string `json:"document_type" binding:"required,oneof=passport national_id drivers_license"`
	DocumentNumber string `json:"document_number" binding:"required,max=50"`
}

func (server *Server) submitKYC(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	userID := payload.UserID

	var req submitKYCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	latest, err := server.store.GetLatestKYCByUser(ctx, userID)
	switch {
	case err == nil:
		if latest.Status == "pending" || latest.Status == "verified" {
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("a verification is already "+latest.Status)))
			return
		}
	case !errors.Is(err, pgx.ErrNoRows):
		handleDBError(ctx, err)
		return
	}

	config, err := utils.LoadConfig("..")
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	enc, err := utils.Encrypt(config.KYCEncryptionKey, req.DocumentNumber)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	k, err := server.store.CreateKYCVerification(ctx, db.CreateKYCVerificationParams{
		UserID:         userID,
		DocumentType:   req.DocumentType,
		DocumentNumber: pgtype.Text{String: enc, Valid: true},
		Status:         "pending",
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	server.audit(ctx, userID, "kyc.submit", "kyc_verification", k.ID,
		map[string]any{"target_user_id": k.UserID})

	ctx.JSON(http.StatusCreated, newKYCSummary(k))
}

// loadPendingKYC checks that the submission exists and is still pending.
// It writes the error response itself and returns ok=false on failure.
func (server *Server) loadPendingKYC(ctx *gin.Context, id int64) (db.KycVerification, bool) {
	existing, err := server.store.GetKYCVerification(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, errorResponse(errors.New("kyc submission not found")))
			return existing, false
		}
		handleDBError(ctx, err)
		return existing, false
	}

	if existing.Status != "pending" {
		ctx.JSON(http.StatusConflict, errorResponse(errors.New("submission already "+existing.Status)))
		return existing, false
	}

	return existing, true
}

func (server *Server) verifyKYC(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	id, ok := parseID(ctx)
	if !ok {
		return
	}

	if _, ok := server.loadPendingKYC(ctx, id); !ok {
		return
	}

	k, err := server.store.VerifyKYC(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// someone else reviewed it between our check and the update
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("submission already reviewed")))
			return
		}
		handleDBError(ctx, err)
		return
	}

	server.audit(ctx, payload.UserID, "kyc.verify", "kyc_verification", k.ID,
		map[string]any{"target_user_id": k.UserID})

	ctx.JSON(http.StatusOK, newAdminKYCResponse(k))
}

type rejectKYCRequest struct {
	RejectionReason string `json:"rejection_reason" binding:"required,max=500"`
}

func (server *Server) rejectKYC(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	id, ok := parseID(ctx)
	if !ok {
		return
	}

	var req rejectKYCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if _, ok := server.loadPendingKYC(ctx, id); !ok {
		return
	}

	k, err := server.store.RejectKYC(ctx, db.RejectKYCParams{
		ID: id,
		RejectionReason: pgtype.Text{
			String: req.RejectionReason,
			Valid:  true,
		},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("submission already reviewed")))
			return
		}
		handleDBError(ctx, err)
		return
	}

	server.audit(ctx, payload.UserID, "kyc.reject", "kyc_verification", k.ID,
		map[string]any{
			"target_user_id": k.UserID,
			"reason":         req.RejectionReason,
		})

	ctx.JSON(http.StatusOK, newAdminKYCResponse(k))
}

type listKYCQuery struct {
	Limit  int32  `form:"limit,default=20"`
	Page   int32  `form:"page,default=1"`
	Status string `form:"status" binding:"omitempty,oneof=pending verified rejected"`
}

func (server *Server) listKYC(ctx *gin.Context) {
	var req listKYCQuery

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	const (
		defaultLimit int32 = 20
		maxLimit     int32 = 100
	)

	if req.Limit <= 0 {
		req.Limit = defaultLimit
	}
	if req.Limit > maxLimit {
		req.Limit = maxLimit
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	offset := (req.Page - 1) * req.Limit

	rows, err := server.store.ListKYCVerifications(ctx, db.ListKYCVerificationsParams{
		Status: req.Status,
		Limit:  req.Limit,
		Offset: offset,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	out := make([]adminKYCResponse, 0, len(rows))
	for _, k := range rows {
		out = append(out, newAdminKYCResponse(k))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":   out,
		"limit":  req.Limit,
		"offset": offset,
	})
}

func (server *Server) getKYCStatus(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	k, err := server.store.GetLatestKYCByUser(ctx, payload.UserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusOK, kycSummary{Status: "not_started"})
			return
		}
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newKYCSummary(k))
}

func (server *Server) audit(ctx *gin.Context, actorID int64, action, entityType string, entityID int64, meta map[string]any) {
	b, _ := json.Marshal(meta)
	err := server.store.CreateAuditLog(ctx, db.CreateAuditLogParams{
		UserID:     pgtype.Int8{Int64: actorID, Valid: true},
		Action:     action,
		EntityType: pgtype.Text{String: entityType, Valid: true},
		EntityID:   pgtype.Int8{Int64: entityID, Valid: true},
		IpAddress:  pgtype.Text{String: ctx.ClientIP(), Valid: true},
		UserAgent:  pgtype.Text{String: ctx.Request.UserAgent(), Valid: true},
		Metadata:   b,
	})
	if err != nil {
		log.Printf("audit log failed: %v", err)
	}
}
