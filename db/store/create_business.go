package store

import (
	"context"

	db "github.com/bank-app/db/sqlc"
)

type CreateBusinessTxResult struct {
	Business       db.Business
	BusinessMember db.BusinessMember
}

func (store *SQLStore) CreateBusinessTx(
	ctx context.Context,
	args db.CreateBusinessParams,
	role string,
) (CreateBusinessTxResult, error) {

	var result CreateBusinessTxResult

	err := store.execTx(ctx, func(q *db.Queries) error {
		var err error

		result.Business, err = q.CreateBusiness(
			ctx,
			db.CreateBusinessParams{
				OwnerID:            args.OwnerID,
				Name:               args.Name,
				RegistrationNumber: args.RegistrationNumber,
				Country:            args.Country,
				Status:             args.Status,
			},
		)

		if err != nil {
			return err
		}

		result.BusinessMember, err = q.AddBusinessMember(
			ctx,
			db.AddBusinessMemberParams{
				BusinessID: result.Business.ID,
				UserID:     result.Business.OwnerID,
				Role:       role,
			},
		)

		if err != nil {
			return err
		}

		return nil
	})

	return result, err
}
