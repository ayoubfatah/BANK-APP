package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type listAccountEntriesURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type listAccountEntriesQuery struct {
	PageID    int64  `form:"page_id,default=1" binding:"min=1"`
	PageSize  int64  `form:"page_size,default=20" binding:"min=1,max=100"`
	From      string `form:"from"`
	To        string `form:"to"`
	MinAmount *int64 `form:"min_amount" binding:"omitempty,min=0"`
	MaxAmount *int64 `form:"max_amount" binding:"omitempty,min=0"`
	Type      string `form:"type"`
}

func (server *Server) ListAccountEntries(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri listAccountEntriesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid account ID")))
		return
	}

	account, err := server.store.GetAccountOwnerAndBalance(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if account.UserID != payload.UserID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("you do not own this account")))
		return
	}

	var query listAccountEntriesQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var fromDate, toDate pgtype.Timestamptz

	if query.From != "" {
		parsed, err := time.Parse(time.RFC3339, query.From)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("from must use RFC3339 format")))
			return
		}
		fromDate = pgtype.Timestamptz{Time: parsed, Valid: true}
	}

	if query.To != "" {
		parsed, err := time.Parse(time.RFC3339, query.To)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("to must use RFC3339 format")))
			return
		}
		toDate = pgtype.Timestamptz{Time: parsed, Valid: true}
	}

	if fromDate.Valid && toDate.Valid && fromDate.Time.After(toDate.Time) {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("from must not be after to")))
		return
	}

	if query.MinAmount != nil && query.MaxAmount != nil &&
		*query.MinAmount > *query.MaxAmount {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("min_amount must not exceed max_amount")))
		return
	}

	if query.PageID-1 > int64((1<<31-1)/query.PageSize) {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("page_id is too large")))
		return
	}

	var minAmount, maxAmount pgtype.Int8

	if query.MinAmount != nil {
		minAmount = pgtype.Int8{Int64: *query.MinAmount, Valid: true}
	}

	if query.MaxAmount != nil {
		maxAmount = pgtype.Int8{Int64: *query.MaxAmount, Valid: true}
	}

	params := db.ListEntriesByAccountIDFilteredParams{
		AccountID: uri.ID,
		FromDate:  fromDate,
		ToDate:    toDate,
		EntryType: pgtype.Text{
			String: query.Type,
			Valid:  query.Type != "",
		},
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Offset:    int32((query.PageID - 1) * query.PageSize),
		PageSize:  int32(query.PageSize),
	}

	entries, err := server.store.ListEntriesByAccountIDFiltered(ctx, params)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"entries":   entries,
		"page_id":   query.PageID,
		"page_size": query.PageSize,
	})
}

// GET /accounts/:id/entries/:entryId

type GetAccountEntryParams struct {
	ID      int64 `uri:"id" binding:"required,min=0"`
	EntryId int64 `uri:"entryId" binding:"required,min=0"`
}

func (server *Server) GetAccountEntry(ctx *gin.Context) {

	payload := getAuthPayload(ctx)

	var params GetAccountEntryParams

	err := ctx.ShouldBindUri(&params)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	entry, err := server.store.GetEntryByAccountID(ctx, db.GetEntryByAccountIDParams{
		ID:        params.EntryId,
		AccountID: params.ID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if entry.UserID != payload.UserID && payload.Role != "admin" {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("This account does't belong to you")))
		return
	}
	ctx.JSON(http.StatusOK, entry)

}

// ADMIN

func (server *Server) AdminListAccountEntries(ctx *gin.Context) {

	var uri listAccountEntriesURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid account ID")))
		return
	}

	var query listAccountEntriesQuery
	if err := ctx.ShouldBindQuery(&query); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var fromDate, toDate pgtype.Timestamptz

	if query.From != "" {
		parsed, err := time.Parse(time.RFC3339, query.From)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("from must use RFC3339 format")))
			return
		}
		fromDate = pgtype.Timestamptz{Time: parsed, Valid: true}
	}

	if query.To != "" {
		parsed, err := time.Parse(time.RFC3339, query.To)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("to must use RFC3339 format")))
			return
		}
		toDate = pgtype.Timestamptz{Time: parsed, Valid: true}
	}

	if fromDate.Valid && toDate.Valid && fromDate.Time.After(toDate.Time) {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("from must not be after to")))
		return
	}

	if query.MinAmount != nil && query.MaxAmount != nil &&
		*query.MinAmount > *query.MaxAmount {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("min_amount must not exceed max_amount")))
		return
	}

	if query.PageID-1 > int64((1<<31-1)/query.PageSize) {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("page_id is too large")))
		return
	}

	var minAmount, maxAmount pgtype.Int8

	if query.MinAmount != nil {
		minAmount = pgtype.Int8{Int64: *query.MinAmount, Valid: true}
	}

	if query.MaxAmount != nil {
		maxAmount = pgtype.Int8{Int64: *query.MaxAmount, Valid: true}
	}

	params := db.ListEntriesByAccountIDFilteredParams{
		AccountID: uri.ID,
		FromDate:  fromDate,
		ToDate:    toDate,
		EntryType: pgtype.Text{
			String: query.Type,
			Valid:  query.Type != "",
		},
		MinAmount: minAmount,
		MaxAmount: maxAmount,
		Offset:    int32((query.PageID - 1) * query.PageSize),
		PageSize:  int32(query.PageSize),
	}

	entries, err := server.store.ListEntriesByAccountIDFiltered(ctx, params)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"entries":   entries,
		"page_id":   query.PageID,
		"page_size": query.PageSize,
	})
}

//

type AdminGetEntriesByReferenceIDRequest struct {
	ReferenceID int64 `json:"reference_id" binding:"required , min=1"`
}

func (server *Server) AdminGetEntriesByReferenceID(ctx *gin.Context) {

	var req AdminGetEntriesByReferenceIDRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	entries, err := server.store.GetEntriesByReferenceID(ctx, pgtype.Int8{
		Int64: req.ReferenceID,
		Valid: true,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"entries": entries,
	})
}

func (server *Server) AdminCheckAccountBalance(ctx *gin.Context) {
	var uri listAccountEntriesURI

	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("invalid account ID")))
		return
	}

	account, err := server.store.GetAccountOwnerAndBalance(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ledgerBalance, err := server.store.GetAccountBalanceFromEntries(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	matches := account.Balance == ledgerBalance

	ctx.JSON(http.StatusOK, gin.H{
		"account_id":      uri.ID,
		"account_balance": account.Balance,
		"ledger_balance":  ledgerBalance,
		"matches":         matches,
	})
}
