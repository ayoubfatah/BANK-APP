package api

import (
	"errors"
	"net/http"

	"github.com/bank-app/db/store"
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
type UpdateAccountDetailsReq struct {
	Type     *string `json:"type" binding:"omitempty"`
	Value    *string `json:"value" binding:"omitempty"`
	Country  *string `json:"country" binding:"omitempty"`
	BankName *string `json:"bank_name" binding:"omitempty"`
}

type UpdateAccountDetailsParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) UpdateAccountDetails(ctx *gin.Context) {

	var param UpdateAccountDetailsParam
	err := ctx.ShouldBindUri(&param)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req UpdateAccountDetailsReq

	err = ctx.ShouldBindJSON(&req)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var args store.UpdateAccountDetailByAccountIDTxParams
	args.AccountID = param.ID
	args.Actor = actorFromCtx(ctx)
	if req.Country != nil {
		args.Country = pgtype.Text{String: *req.Country, Valid: true}
	}
	if req.BankName != nil {
		args.BankName = pgtype.Text{String: *req.BankName, Valid: true}
	}
	if req.Type != nil {
		args.Type = *req.Type

	}

	if req.Value != nil {
		args.Value = *req.Value

	}

	updatedAccountDetails, err := server.store.UpdateAccountDetailByAccountIDTx(ctx, args)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, updatedAccountDetails)

}
