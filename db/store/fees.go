package store

import db "github.com/bank-app/db/sqlc"

func CalculateFee(rule db.FeeRule, amount int64) int64 {
	var fee int64

	switch rule.FeeType {
	case "flat":
		fee = rule.FlatAmount
	case "percentage":
		fee = (amount*int64(rule.PercentageBps) + 5000) / 10000
	}

	if fee < rule.MinFee {
		fee = rule.MinFee
	}
	if rule.MaxFee.Valid && fee > rule.MaxFee.Int64 {
		fee = rule.MaxFee.Int64
	}

	return fee
}
