package api

import (
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
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

func (server *Server) GetAccounts(ctx *gin.Context){
	payload := getAuthPayload(ctx)


	accounts , err  := server.store.ListAccountsByUserID(ctx ,payload.UserID)
	if err != nil {
		handleDBError(ctx,err)
		return
	}


	ctx.JSON(http.StatusOK , accounts)
}



type GetAccountParams struct {
 	ID   int64 `uri:"id" binding:"required,min=1"`
}


func (server *Server) GetAccount(ctx *gin.Context){
	payload := getAuthPayload(ctx)
	   var accountParams GetAccountParams
    if err := ctx.ShouldBindUri(&accountParams); err != nil {
      ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
      return
    }

	account , err :=  server.store.GetAccountByIDAndUserID(ctx ,db.GetAccountByIDAndUserIDParams{
		ID: accountParams.ID,
		UserID: payload.UserID,
	})
	if err != nil{
		handleDBError(ctx ,err)
		return
	}

	ctx.JSON(http.StatusOK ,account)
}
func (server *Server) FreezeAccount(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var accountParams GetAccountParams
	if err := ctx.ShouldBindUri(&accountParams); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := server.store.FreezeAccount(ctx, db.FreezeAccountParams{
		ID:     accountParams.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"message": "Account frozen successfully",
	})
}
