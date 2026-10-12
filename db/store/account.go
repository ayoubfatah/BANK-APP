package store

import (
	"context"
	"fmt"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgtype"
)


type CreateAccountTxArguments struct {
	User db.User
	CurrencyCode string 
	BankingPartnerID int64
	Balance int64
}


type CreateAccountTxResult struct {
	Account db.Account
	AccountDetails db.AccountDetail
	BankPartner db.BankingPartner
}



func (store *Store) CreateAccountTx(ctx context.Context , args CreateAccountTxArguments) (CreateAccountTxResult,error) {

	var result  CreateAccountTxResult
	err := store.execTx(ctx,func(q *db.Queries) error {
	var err error
	currency , err := q.GetCurrency(ctx , args.CurrencyCode)

	if err != nil {
			return err
	}

	result.Account  , err  =  q.CreateAccount(ctx,db.CreateAccountParams{
		UserID: args.User.ID,
		Currency: args.CurrencyCode,
		Balance: args.Balance,
		Status: "active",
	})

	if err != nil {
			return err
	}
	result.BankPartner ,err = q.GetBankingPartner(
		ctx,
		args.BankingPartnerID,
	)

	if err != nil {
			return err
	}	
	if result.BankPartner.CurrencyCode != currency.Code{
    return fmt.Errorf("wrong currency")
	}
	
	if !result.BankPartner.IsActive {
    return fmt.Errorf("banking partner is inactive")
	}
	
	value , err := utils.GenerateAccountDetail(result.BankPartner.DetailType , result.BankPartner.CountryCode)

	if err != nil {
			return err
	}	


	result.AccountDetails , err = q.CreateAccountDetail(ctx ,db.CreateAccountDetailParams{
		AccountID: result.Account.ID,
		Type: result.BankPartner.DetailType,
		BankName: pgtype.Text{
			String: result.BankPartner.ProviderName,
			Valid: true,
		},
		Country: pgtype.Text{
			String: result.BankPartner.CountryCode,
			Valid: true,
		},
		Value:value})
	
	if err != nil {
			return err
	}	

	return  err	
})


return result, err


}