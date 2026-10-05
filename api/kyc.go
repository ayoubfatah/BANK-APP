package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)


type kycSummary struct {
	Status          string     `json:"status" binding:"oneof=not_started pending verified rejected"` // not_started | pending | verified | rejected
	DocumentType    *string    `json:"document_type"`
	RejectionReason *string    `json:"rejection_reason"`
	VerifiedAt      *time.Time `json:"verified_at"`
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
 
func (server *Server) SubmitKYC(ctx *gin.Context) {
	userID := int64(1)
 
	var req submitKYCRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
 
	// no new submission while one is pending or already verified
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
 
	k, err := server.store.CreateKYCVerification(ctx, db.CreateKYCVerificationParams{
		UserID:         userID,
		DocumentType:   req.DocumentType,
		DocumentNumber: pgtype.Text{String: req.DocumentNumber, Valid: true},
		Status:         "pending",
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, newKYCSummary(k))
}
 


func (server *Server) GetKYCStatus(ctx *gin.Context) {
	k, err := server.store.GetLatestKYCByUser(ctx, int64(1))
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

