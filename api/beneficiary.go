package api

import (
	"net/http"
	"strings"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type beneficiaryResponse struct {
	ID            int64     `json:"id"`
	Name          string    `json:"name"`
	Nickname      string    `json:"nickname,omitempty"`
	BankName      string    `json:"bank_name"`
	AccountNumber string    `json:"account_number,omitempty"`
	Iban          string    `json:"iban,omitempty"`
	Bic           string    `json:"bic,omitempty"`
	RoutingNumber string    `json:"routing_number,omitempty"`
	Country       string    `json:"country"`
	Currency      string    `json:"currency"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newBeneficiaryResponse(b db.Beneficiary) beneficiaryResponse {
	return beneficiaryResponse{
		ID:            b.ID,
		Name:          b.Name,
		Nickname:      b.Nickname.String,
		BankName:      b.BankName.String,
		AccountNumber: b.AccountNumber.String,
		Iban:          b.Iban.String,
		Bic:           b.Bic.String,
		RoutingNumber: b.RoutingNumber.String,
		Country:       b.Country,
		Currency:      b.Currency.String,
		CreatedAt:     b.CreatedAt.Time,
		UpdatedAt:     b.UpdatedAt.Time,
	}
}

func beneTrim(s string) string  { return strings.TrimSpace(s) }
func beneUpper(s string) string { return strings.ToUpper(strings.TrimSpace(s)) }
func beneIban(s string) string {
	return strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(s), " ", ""))
}

func beneText(s string) pgtype.Text {
	return pgtype.Text{String: s, Valid: s != ""}
}

func benePtrText(s *string, clean func(string) string) pgtype.Text {
	if s == nil {
		return pgtype.Text{}
	}
	return pgtype.Text{String: clean(*s), Valid: true}
}

type beneficiaryURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type createBeneficiaryRequest struct {
	Name          string `json:"name" binding:"required,min=2,max=100"`
	Nickname      string `json:"nickname" binding:"omitempty,max=50"`
	BankName      string `json:"bank_name" binding:"required,min=2,max=100"`
	AccountNumber string `json:"account_number" binding:"required_without=Iban,omitempty,min=4,max=34"`
	Iban          string `json:"iban" binding:"required_without=AccountNumber,omitempty,min=15,max=42"`
	Bic           string `json:"bic" binding:"omitempty,min=8,max=11"`
	RoutingNumber string `json:"routing_number" binding:"omitempty,max=20"`
	Country       string `json:"country" binding:"required,len=2"`
	Currency      string `json:"currency" binding:"required,len=3"`
}

// POST /beneficiaries
func (server *Server) CreateBeneficiary(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req createBeneficiaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	beneficiary, err := server.store.CreateBeneficiary(ctx, db.CreateBeneficiaryParams{
		UserID:        payload.UserID,
		Name:          beneTrim(req.Name),
		Nickname:      beneText(beneTrim(req.Nickname)),
		AccountNumber: beneText(beneTrim(req.AccountNumber)),
		Iban:          beneText(beneIban(req.Iban)),
		Bic:           beneText(beneUpper(req.Bic)),
		RoutingNumber: beneText(beneTrim(req.RoutingNumber)),
		BankName:      beneText(req.BankName),
		Country:       beneUpper(req.Country),
		Currency:      beneText(req.Currency),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newBeneficiaryResponse(beneficiary))
}

type listBeneficiariesRequest struct {
	Search string `form:"search" binding:"omitempty,max=100"`
}

// GET /beneficiaries
func (server *Server) ListBeneficiaries(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req listBeneficiariesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	rows, err := server.store.ListBeneficiariesByUserID(ctx, db.ListBeneficiariesByUserIDParams{
		UserID: payload.UserID,
		Search: beneText(beneTrim(req.Search)),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	beneficiaries := make([]beneficiaryResponse, 0, len(rows))
	for _, row := range rows {
		beneficiaries = append(beneficiaries, newBeneficiaryResponse(row))
	}

	ctx.JSON(http.StatusOK, beneficiaries)
}

// GET /beneficiaries/:id
func (server *Server) GetBeneficiary(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri beneficiaryURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	beneficiary, err := server.store.GetBeneficiary(ctx, db.GetBeneficiaryParams{
		ID:     uri.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newBeneficiaryResponse(beneficiary))
}

type updateBeneficiaryRequest struct {
	Name          *string `json:"name" binding:"omitempty,min=2,max=100"`
	Nickname      *string `json:"nickname" binding:"omitempty,max=50"`
	BankName      *string `json:"bank_name" binding:"omitempty,min=2,max=100"`
	AccountNumber *string `json:"account_number" binding:"omitempty,min=4,max=34"`
	Iban          *string `json:"iban" binding:"omitempty,min=15,max=42"`
	Bic           *string `json:"bic" binding:"omitempty,min=8,max=11"`
	RoutingNumber *string `json:"routing_number" binding:"omitempty,max=20"`
	Country       *string `json:"country" binding:"omitempty,len=2"`
	Currency      *string `json:"currency" binding:"omitempty,len=3"`
}

func (r updateBeneficiaryRequest) isEmpty() bool {
	return r.Name == nil && r.Nickname == nil && r.BankName == nil &&
		r.AccountNumber == nil && r.Iban == nil && r.Bic == nil &&
		r.RoutingNumber == nil && r.Country == nil && r.Currency == nil
}

// PATCH /beneficiaries/:id
func (server *Server) UpdateBeneficiary(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri beneficiaryURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateBeneficiaryRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.isEmpty() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}  

	beneficiary, err := server.store.UpdateBeneficiary(ctx, db.UpdateBeneficiaryParams{
		ID:            uri.ID,
		UserID:        payload.UserID,
		Name:          benePtrText(req.Name, beneTrim),
		Nickname:      benePtrText(req.Nickname, beneTrim),
		BankName:      benePtrText(req.BankName, beneTrim),
		AccountNumber: benePtrText(req.AccountNumber, beneTrim),
		Iban:          benePtrText(req.Iban, beneIban),
		Bic:           benePtrText(req.Bic, beneUpper),
		RoutingNumber: benePtrText(req.RoutingNumber, beneTrim),
		Country:       benePtrText(req.Country, beneUpper),
		Currency:      benePtrText(req.Currency, beneUpper),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newBeneficiaryResponse(beneficiary))
}

// DELETE /beneficiaries/:id
func (server *Server) DeleteBeneficiary(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri beneficiaryURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	rowsAffected, err := server.store.DeleteBeneficiary(ctx, db.DeleteBeneficiaryParams{
		ID:     uri.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if rowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Beneficiary not found"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Beneficiary deleted successfully"})
}
