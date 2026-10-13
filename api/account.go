package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type CreateAccountRequest struct {
	CurrencyCode     string `json:"currency_code" binding:"required,min=3,max=3"`
	BankingPartnerID int64  `json:"banking_partner_id" binding:"min=1"`
}

type createAccountResponse struct {
	ID               int64     `json:"id"`
	UserID           int64     `json:"user_id"`
	CurrencyCode     string    `json:"currency_code"`
	BankingPartnerID int64     `json:"banking_partner_id"`
	Balance          int64     `json:"balance"`
	CreatedAt        time.Time `json:"created_at"`
}

func newCreateAccountResponse(
	result store.CreateAccountTxResult,
) createAccountResponse {
	return createAccountResponse{
		ID:               result.Account.ID,
		UserID:           result.Account.UserID,
		CurrencyCode:     result.Account.Currency,
		BankingPartnerID: result.BankPartner.ID,
		Balance:          result.Account.Balance,
		CreatedAt:        result.Account.CreatedAt.Time,
	}
}

// POST /accounts
func (server *Server) CreateAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var req CreateAccountRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	createAccountResult, err := server.store.CreateAccountTx(ctx,
		store.CreateAccountTxArguments{
			UserID:           payload.UserID,
			CurrencyCode:     req.CurrencyCode,
			BankingPartnerID: req.BankingPartnerID,
			Balance:          0,
		})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newCreateAccountResponse(createAccountResult))

}

// GET /accounts
func (server *Server) GetAccounts(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	accounts, err := server.store.ListAccountsByUserID(ctx, payload.UserID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, accounts)
}

type GetAccountParams struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GET /accounts/:id
func (server *Server) GetAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var accountParams GetAccountParams
	if err := ctx.ShouldBindUri(&accountParams); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     accountParams.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type FreezeAccountParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// POST /accounts/:id/freeze
func (server *Server) FreezeAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var accountParams FreezeAccountParam
	if err := ctx.ShouldBindUri(&accountParams); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := server.store.GetAccount(ctx, accountParams.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	// Customers can only freeze their own accounts.
	if payload.UserID != account.UserID && payload.Role != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "You cannot freeze an account that doesn't belong to you",
		})
		return
	}

	rowsAffected, err := server.store.FreezeAccountTx(
		ctx,
		store.FreezeAccountTxParams{
			AccountID: account.ID,
			UserID:    account.UserID,
			ActorID:   payload.UserID,
			ActorRole: payload.Role,
			IPAddress: ctx.ClientIP(),
			UserAgent: ctx.Request.UserAgent(),
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "Account is already frozen or cannot be frozen",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account frozen successfully",
	})
}

type UnFreezeAccountParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// POST /accounts/:id/unfreeze
func (server *Server) UnFreezeAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var accountParams UnFreezeAccountParam
	if err := ctx.ShouldBindUri(&accountParams); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	account, err := server.store.GetAccount(ctx, accountParams.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	// Customers can only unfreeze their own accounts.
	if payload.UserID != account.UserID && payload.Role != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "You cannot unfreeze an account that doesn't belong to you",
		})
		return
	}

	// Customers cannot unfreeze an account frozen by another user.
	if account.FreezedBy.Valid &&
		account.FreezedBy.Int64 != payload.UserID &&
		payload.Role != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "You cannot unfreeze an account frozen by another user",
		})
		return
	}

	rowsAffected, err := server.store.UnFreezeAccountTx(
		ctx,
		store.UnFreezeAccountTxParams{
			AccountID: account.ID,
			UserID:    account.UserID,
			ActorID:   payload.UserID,
			ActorRole: payload.Role,
			IPAddress: ctx.ClientIP(),
			UserAgent: ctx.Request.UserAgent(),
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "Account is already unfrozen or cannot be unfrozen",
		})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account unfrozen successfully",
	})
}

type CloseAccountParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// POST /accounts/:id/close
func (server *Server) CloseAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var accountParams CloseAccountParam
	if err := ctx.ShouldBindUri(&accountParams); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, accountParams.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if payload.UserID != account.UserID && payload.Role != "admin" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "You cannot close an account that doesn't belong to you",
		})
		return
	}

	rowsAffected, err := server.store.CloseAccountTx(
		ctx,
		store.CloseAccountTxParams{
			AccountID: account.ID,
			UserID:    account.UserID, // Account owner's ID
			ActorID:   payload.UserID, // Person performing the action
			ActorRole: payload.Role,
			IPAddress: ctx.ClientIP(),
			UserAgent: ctx.Request.UserAgent(),
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, errorResponse(
			errors.New("Account is already closed or cannot be closed"),
		))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account closed successfully",
	})
}

type adminListAccountsRequest struct {
	UserID   *int64  `form:"user_id" binding:"omitempty,min=1"`
	Currency *string `form:"currency" binding:"omitempty,len=3"`
	Status   *string `form:"status" binding:"omitempty,oneof=active frozen closed"`
	PageID   int32   `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32   `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GET /admin/accounts
func (server *Server) AdminListAccounts(ctx *gin.Context) {
	var req adminListAccountsRequest
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

	arg := db.ListAccountsAdminParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}
	if req.UserID != nil {
		arg.UserID = pgtype.Int8{Int64: *req.UserID, Valid: true}
	}
	if req.Currency != nil {
		arg.Currency = pgtype.Text{String: *req.Currency, Valid: true}
	}
	if req.Status != nil {
		arg.Status = pgtype.Text{String: *req.Status, Valid: true}
	}

	accounts, err := server.store.ListAccountsAdmin(ctx, arg)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":   req.PageID,
		"page_size": req.PageSize,
		"accounts":  accounts,
	})
}

type adminAccountURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GET /admin/accounts/:id
func (server *Server) AdminGetAccount(ctx *gin.Context) {
	var uri adminAccountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, account)
}

type adminActionRequest struct {
	Reason string `json:"reason" binding:"required,min=5,max=500"`
}

// POST /admin/accounts/:id/freeze
func (server *Server) AdminFreezeAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri adminAccountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req adminActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rowsAffected, err := server.store.FreezeAccountTx(ctx, store.FreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    account.UserID,
		ActorID:   payload.UserID,
		ActorRole: payload.Role,
		IPAddress: ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Account is already frozen or cannot be frozen"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Account frozen successfully"})
}

// POST /admin/accounts/:id/unfreeze
func (server *Server) AdminUnfreezeAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri adminAccountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req adminActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rowsAffected, err := server.store.UnFreezeAccountTx(ctx, store.UnFreezeAccountTxParams{
		AccountID: account.ID,
		UserID:    account.UserID,
		ActorID:   payload.UserID,
		ActorRole: payload.Role,
		IPAddress: ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Account is already unfrozen or cannot be unfrozen"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Account unfrozen successfully"})
}

// POST /admin/accounts/:id/close
func (server *Server) AdminCloseAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri adminAccountURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req adminActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccount(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rowsAffected, err := server.store.CloseAccountTx(ctx, store.CloseAccountTxParams{
		AccountID: account.ID,
		UserID:    account.UserID,
		ActorID:   payload.UserID,
		ActorRole: payload.Role,
		IPAddress: ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if rowsAffected == 0 {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Account is already closed or cannot be closed"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Account closed successfully"})
}
