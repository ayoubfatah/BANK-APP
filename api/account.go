package api

import (
	"net/http"
	"time"

	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
)


type CreateAccountRequest struct {
	CurrencyCode string `json:"currency_code" binding:"required,min=3,max=3"`
	BankingPartnerID int64 `json:"banking_partner_id" binding:"min=1"`
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
		ID: result.Account.ID,
		UserID: result.Account.UserID,
		CurrencyCode: result.Account.Currency,
		BankingPartnerID: result.BankPartner.ID,
		Balance: result.Account.Balance,
		CreatedAt: result.Account.CreatedAt.Time,
	}
}


func (server *Server) CreateAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var req CreateAccountRequest 

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	createAccountResult , err := server.store.CreateAccountTx(ctx,
	store.CreateAccountTxArguments{
		UserID: payload.UserID,
		CurrencyCode: req.CurrencyCode,
		BankingPartnerID: req.BankingPartnerID,
		Balance: 0,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, newCreateAccountResponse(createAccountResult))

}