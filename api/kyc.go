package api

import (
	"encoding/json"
	"errors"
	"fmt"
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
 
func (server *Server) SubmitKYC(ctx *gin.Context) {
	userID := int64(1)
 
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
 
	config, err := utils.LoadConfig("../../")

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":err.Error(),
		})
		return 
	}
	enc, err := utils.Encrypt(config.KYCEncryptionKey, req.DocumentNumber)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":err.Error(),
		})
		return 
	}
	fmt.Println("THIS IS THE ENCRYPTED FILE")
	fmt.Println(enc)

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

	server.audit(ctx,userID, "kyc.verify", "kyc_verification", k.ID,
	map[string]any{"target_user_id": k.UserID})

	
	ctx.JSON(http.StatusCreated, newKYCSummary(k))
}
 



func (server *Server) verifyKYC(ctx *gin.Context) {
	id, ok := parseID(ctx)
	if !ok {
		return
	}
 
	k, err := server.store.VerifyKYC(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) { 
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("submission not found or already reviewed")))
			return
		}
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newAdminKYCResponse(k))
}



type rejectKYCRequest struct {
	RejectionReason string `json:"rejection_reason" binding:"required,max=500"`
}
func (server *Server) rejectKYC(ctx *gin.Context){
	var req rejectKYCRequest

	if err := ctx.ShouldBindJSON(&req);err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return 
	}

	id, ok := parseID(ctx)
	if !ok {
		return
	}
	k  , err := server.store.RejectKYC(ctx , db.RejectKYCParams{
		ID: id,
		RejectionReason:  pgtype.Text{
			String: req.RejectionReason,
			Valid: true,
		} ,
	})
   
	if err != nil {

			if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusConflict, errorResponse(errors.New("submission not found or already reviewed"))) 
		    return 

		}
		handleDBError(ctx, err)
		return 
	}
	ctx.JSON(http.StatusOK, newAdminKYCResponse(k))
}

// list 

type listKYCQuery struct {
	Limit  int32  `form:"limit,default=20"`
	Offset int32  `form:"offset,default=0"`
	Status string `form:"status" binding:"omitempty,oneof=pending verified rejected"`
}

func (server *Server) ListKYC(ctx *gin.Context) {
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

    if req.Offset < 0 {
        req.Offset = 0
    }
	
	rows, err := server.store.ListKYCVerifications(ctx, db.ListKYCVerificationsParams{
		Status: req.Status,
		Limit:  req.Limit,
		Offset: req.Offset,
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
		"offset": req.Offset,
	})
}





func (server *Server) getKYCStatus(ctx *gin.Context) {
	userID := int64(1) // TODO: from token

	k, err := server.store.GetLatestKYCByUser(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			// no KYC yet is a normal state, not an error
			ctx.JSON(http.StatusOK, kycSummary{Status: "not_started"})
			return
		}
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newKYCSummary(k))
}


func (server *Server) audit(ctx *gin.Context, actorID int64, action, entityType string, entityID int64, meta map[string]any) {
	fmt.Println("running")
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