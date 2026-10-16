package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func feePercentage(bps int32) string {
	return fmt.Sprintf("%.2f%%", float64(bps)/100)
}

func validateFeeRule(
	feeType string,
	flatAmount int64,
	percentageBps int32,
	minFee int64,
	maxFee pgtype.Int8,
) string {
	switch feeType {
	case "flat":
		if flatAmount <= 0 {
			return "flat_amount must be greater than 0 for a flat fee"
		}
		if percentageBps != 0 {
			return "percentage_bps must be 0 for a flat fee"
		}
	case "percentage":
		if percentageBps <= 0 {
			return "percentage_bps must be greater than 0 for a percentage fee"
		}
		if flatAmount != 0 {
			return "flat_amount must be 0 for a percentage fee"
		}
	}
	if maxFee.Valid && maxFee.Int64 < minFee {
		return "max_fee cannot be less than min_fee"
	}
	return ""
}

type feeScheduleItem struct {
	OperationType string `json:"operation_type"`
	Currency      string `json:"currency"`
	FeeType       string `json:"fee_type"`
	FlatAmount    int64  `json:"flat_amount"`
	Percentage    string `json:"percentage"`
	MinFee        int64  `json:"min_fee"`
	MaxFee        *int64 `json:"max_fee,omitempty"`
}

func newFeeScheduleItem(r db.FeeRule) feeScheduleItem {
	item := feeScheduleItem{
		OperationType: r.OperationType,
		Currency:      r.Currency,
		FeeType:       r.FeeType,
		FlatAmount:    r.FlatAmount,
		Percentage:    feePercentage(r.PercentageBps),
		MinFee:        r.MinFee,
	}
	if r.MaxFee.Valid {
		maxFee := r.MaxFee.Int64
		item.MaxFee = &maxFee
	}
	return item
}

type feeRuleResponse struct {
	ID            int64     `json:"id"`
	OperationType string    `json:"operation_type"`
	Currency      string    `json:"currency"`
	FeeType       string    `json:"fee_type"`
	FlatAmount    int64     `json:"flat_amount"`
	PercentageBps int32     `json:"percentage_bps"`
	Percentage    string    `json:"percentage"`
	MinFee        int64     `json:"min_fee"`
	MaxFee        *int64    `json:"max_fee,omitempty"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func newFeeRuleResponse(r db.FeeRule) feeRuleResponse {
	res := feeRuleResponse{
		ID:            r.ID,
		OperationType: r.OperationType,
		Currency:      r.Currency,
		FeeType:       r.FeeType,
		FlatAmount:    r.FlatAmount,
		PercentageBps: r.PercentageBps,
		Percentage:    feePercentage(r.PercentageBps),
		MinFee:        r.MinFee,
		IsActive:      r.IsActive,
		CreatedAt:     r.CreatedAt.Time,
		UpdatedAt:     r.UpdatedAt.Time,
	}
	if r.MaxFee.Valid {
		maxFee := r.MaxFee.Int64
		res.MaxFee = &maxFee
	}
	return res
}

type feeResponse struct {
	ID         int64     `json:"id"`
	TransferID int64     `json:"transfer_id"`
	Type       string    `json:"type"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	CreatedAt  time.Time `json:"created_at"`
}

func newFeeResponse(f db.Fee) feeResponse {
	return feeResponse{
		ID:         f.ID,
		TransferID: f.TransferID.Int64,
		Type:       f.Type,
		Amount:     f.Amount,
		Currency:   f.Currency,
		CreatedAt:  f.CreatedAt.Time,
	}
}

func newFeeResponses(rows []db.Fee) []feeResponse {
	fees := make([]feeResponse, 0, len(rows))
	for _, row := range rows {
		fees = append(fees, newFeeResponse(row))
	}
	return fees
}

// GET /fees/schedule
func (server *Server) GetFeeSchedule(ctx *gin.Context) {
	rules, err := server.store.ListActiveFeeRules(ctx)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	schedule := make([]feeScheduleItem, 0, len(rules))
	for _, rule := range rules {
		schedule = append(schedule, newFeeScheduleItem(rule))
	}

	ctx.JSON(http.StatusOK, schedule)
}

// GET /transfers/:id/fees
func (server *Server) ListTransferFees(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri transferURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	transfer, err := server.store.GetTransfer(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	owns, err := server.userOwnsTransfer(ctx, payload.UserID, transfer)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if !owns {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	fees := []feeResponse{}
	var totalFees int64

	// only the sender pays the fee, so only the sender sees it
	_, err = server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     transfer.FromAccountID.Int64,
		UserID: payload.UserID,
	})
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		handleDBError(ctx, err)
		return
	}
	if err == nil {
		rows, err := server.store.ListFeesByTransferID(ctx, pgtype.Int8{Int64: transfer.ID, Valid: true})
		if err != nil {
			handleDBError(ctx, err)
			return
		}
		totalFees, err = server.store.GetTransferFeesTotal(ctx, pgtype.Int8{Int64: transfer.ID, Valid: true})
		if err != nil {
			handleDBError(ctx, err)
			return
		}
		fees = newFeeResponses(rows)
	}

	ctx.JSON(http.StatusOK, gin.H{
		"transfer_id": transfer.ID,
		"currency":    transfer.Currency,
		"total_fees":  totalFees,
		"fees":        fees,
	})
}

type adminListFeeRulesRequest struct {
	OperationType *string `form:"operation_type" binding:"omitempty,oneof=transfer conversion card"`
	Currency      *string `form:"currency" binding:"omitempty,len=3"`
	IsActive      *bool   `form:"is_active"`
}

// GET /admin/fee-rules
func (server *Server) AdminListFeeRules(ctx *gin.Context) {
	var req adminListFeeRulesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var arg db.ListFeeRulesParams
	if req.OperationType != nil {
		arg.OperationType = pgtype.Text{String: *req.OperationType, Valid: true}
	}
	if req.Currency != nil {
		arg.Currency = pgtype.Text{String: strings.ToUpper(*req.Currency), Valid: true}
	}
	if req.IsActive != nil {
		arg.IsActive = pgtype.Bool{Bool: *req.IsActive, Valid: true}
	}

	rows, err := server.store.ListFeeRules(ctx, arg)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rules := make([]feeRuleResponse, 0, len(rows))
	for _, row := range rows {
		rules = append(rules, newFeeRuleResponse(row))
	}

	ctx.JSON(http.StatusOK, rules)
}

type createFeeRuleRequest struct {
	OperationType string `json:"operation_type" binding:"required,oneof=transfer conversion card"`
	Currency      string `json:"currency" binding:"required,len=3"`
	FeeType       string `json:"fee_type" binding:"required,oneof=flat percentage"`
	FlatAmount    int64  `json:"flat_amount" binding:"min=0"`
	PercentageBps int32  `json:"percentage_bps" binding:"min=0,max=10000"`
	MinFee        int64  `json:"min_fee" binding:"min=0"`
	MaxFee        *int64 `json:"max_fee" binding:"omitempty,min=0"`
}

// POST /admin/fee-rules
func (server *Server) AdminCreateFeeRule(ctx *gin.Context) {
	var req createFeeRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var maxFee pgtype.Int8
	if req.MaxFee != nil {
		maxFee = pgtype.Int8{Int64: *req.MaxFee, Valid: true}
	}

	if msg := validateFeeRule(req.FeeType, req.FlatAmount, req.PercentageBps, req.MinFee, maxFee); msg != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	rule, err := server.store.CreateFeeRule(ctx, db.CreateFeeRuleParams{
		OperationType: req.OperationType,
		Currency:      strings.ToUpper(req.Currency),
		FeeType:       req.FeeType,
		FlatAmount:    req.FlatAmount,
		PercentageBps: req.PercentageBps,
		MinFee:        req.MinFee,
		MaxFee:        maxFee,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newFeeRuleResponse(rule))
}

type updateFeeRuleRequest struct {
	FeeType       *string `json:"fee_type" binding:"omitempty,oneof=flat percentage"`
	FlatAmount    *int64  `json:"flat_amount" binding:"omitempty,min=0"`
	PercentageBps *int32  `json:"percentage_bps" binding:"omitempty,min=0,max=10000"`
	MinFee        *int64  `json:"min_fee" binding:"omitempty,min=0"`
	MaxFee        *int64  `json:"max_fee" binding:"omitempty,min=0"`
	IsActive      *bool   `json:"is_active"`
}

func (r updateFeeRuleRequest) isEmpty() bool {
	return r.FeeType == nil && r.FlatAmount == nil && r.PercentageBps == nil &&
		r.MinFee == nil && r.MaxFee == nil && r.IsActive == nil
}

type feeRuleURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// PATCH /admin/fee-rules/:id
func (server *Server) AdminUpdateFeeRule(ctx *gin.Context) {
	var uri feeRuleURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req updateFeeRuleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.isEmpty() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "No fields to update"})
		return
	}

	rule, err := server.store.GetFeeRule(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	feeType := rule.FeeType
	flatAmount := rule.FlatAmount
	percentageBps := rule.PercentageBps
	minFee := rule.MinFee
	maxFee := rule.MaxFee
	isActive := rule.IsActive

	if req.FeeType != nil {
		feeType = *req.FeeType
	}
	if req.FlatAmount != nil {
		flatAmount = *req.FlatAmount
	}
	if req.PercentageBps != nil {
		percentageBps = *req.PercentageBps
	}
	if req.MinFee != nil {
		minFee = *req.MinFee
	}
	if req.MaxFee != nil {
		maxFee = pgtype.Int8{Int64: *req.MaxFee, Valid: true}
	}
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	if msg := validateFeeRule(feeType, flatAmount, percentageBps, minFee, maxFee); msg != "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return
	}

	updated, err := server.store.UpdateFeeRule(ctx, db.UpdateFeeRuleParams{
		ID:            rule.ID,
		FeeType:       pgtype.Text{String: feeType, Valid: true},
		FlatAmount:    pgtype.Int8{Int64: flatAmount, Valid: true},
		PercentageBps: pgtype.Int4{Int32: percentageBps, Valid: true},
		MinFee:        pgtype.Int8{Int64: minFee, Valid: true},
		MaxFee:        maxFee,
		IsActive:      pgtype.Bool{Bool: isActive, Valid: true},
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newFeeRuleResponse(updated))
}

type adminListFeesRequest struct {
	Type     *string `form:"type" binding:"omitempty,oneof=transfer conversion card"`
	Currency *string `form:"currency" binding:"omitempty,len=3"`
	From     *string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To       *string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	PageID   int32   `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32   `form:"page_size" binding:"omitempty,min=1,max=100"`
}

type feeTotalResponse struct {
	Type        string `json:"type"`
	Currency    string `json:"currency"`
	FeeCount    int64  `json:"fee_count"`
	TotalAmount int64  `json:"total_amount"`
}

// GET /admin/fees
func (server *Server) AdminListFees(ctx *gin.Context) {
	var req adminListFeesRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.PageID == 0 {
		req.PageID = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 20
	}

	var feeType, currency pgtype.Text
	var fromDate, toDate pgtype.Timestamptz

	if req.Type != nil {
		feeType = pgtype.Text{String: *req.Type, Valid: true}
	}
	if req.Currency != nil {
		currency = pgtype.Text{String: strings.ToUpper(*req.Currency), Valid: true}
	}
	if req.From != nil {
		from, err := time.Parse("2006-01-02", *req.From)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "from must be YYYY-MM-DD"})
			return
		}
		fromDate = pgtype.Timestamptz{Time: from, Valid: true}
	}
	if req.To != nil {
		to, err := time.Parse("2006-01-02", *req.To)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "to must be YYYY-MM-DD"})
			return
		}
		toDate = pgtype.Timestamptz{Time: to.AddDate(0, 0, 1), Valid: true}
	}
	if fromDate.Valid && toDate.Valid && !fromDate.Time.Before(toDate.Time) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "from must not be after to"})
		return
	}

	rows, err := server.store.ListFeesAdmin(ctx, db.ListFeesAdminParams{
		FeeType:  feeType,
		Currency: currency,
		FromDate: fromDate,
		ToDate:   toDate,
		Limit:    req.PageSize,
		Offset:   (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	sums, err := server.store.SumFeesAdmin(ctx, db.SumFeesAdminParams{
		FeeType:  feeType,
		Currency: currency,
		FromDate: fromDate,
		ToDate:   toDate,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	totals := make([]feeTotalResponse, 0, len(sums))
	for _, s := range sums {
		totals = append(totals, feeTotalResponse{
			Type:        s.Type,
			Currency:    s.Currency,
			FeeCount:    s.FeeCount,
			TotalAmount: s.TotalAmount,
		})
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":   req.PageID,
		"page_size": req.PageSize,
		"totals":    totals,
		"fees":      newFeeResponses(rows),
	})
}
