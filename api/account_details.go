package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type GetAccountDetailsParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) GetAccountDetails(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var param GetAccountDetailsParam

	err := ctx.ShouldBindUri(&param)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccountDetailsWithUserID(ctx, param.ID)
	if account.UserID != payload.UserID && payload.Role != "admin" {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("this account details doesn't belong to you ")))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": account,
	})

}

// POST /admin/accounts/:id/details
// PATCH /admin/accounts/:id/details/:detailId
// DELETE /admin/accounts/:id/details/:detailId

type UpdateAccountDetailsParam struct {
	ID       int64 `uri:"id" binding:"required,min=1"`
	DetailID int64 `uri:"detailId" binding:"required,min=1"`
}

type UpdateAccountDetailsReq struct {
	Type     string  `json:"type" binding:"required"`
	Value    string  `json:"value" binding:"required"`
	Country  *string `json:"country" binding:"omitempty"`
	BankName *string `json:"bank_name" binding:"omitempty"`
}

func (server *Server) UpdateAccountDetails(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var param UpdateAccountDetailsParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req UpdateAccountDetailsReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	args := store.UpdateAccountDetailTxParams{
		AccountID: param.ID,
		DetailID:  param.DetailID,
		Type:      req.Type,
		Value:     req.Value,
		AdminID:   payload.UserID,
		IPAddress: ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
	}

	if req.Country != nil {
		args.Country = pgtype.Text{String: *req.Country, Valid: true}
	}
	if req.BankName != nil {
		args.BankName = pgtype.Text{String: *req.BankName, Valid: true}
	}

	rowsAffected, err := server.store.UpdateAccountDetailTx(ctx, args)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if rowsAffected == 0 {
		ctx.JSON(http.StatusNotFound, errorResponse(errors.New("account detail not found")))
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "account detail updated"})
}
