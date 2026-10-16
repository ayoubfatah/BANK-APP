package api

import (
	"net/http"
	"strings"

	db "github.com/bank-app/db/sqlc"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"
)

type exchangeRateResponse struct {
	ID   int64  `json:"id"`
	From string `json:"from"`
	To   string `json:"to"`
	Rate string `json:"rate"`
}

func newExchangeRateResponse(
	id int64,
	from, to string,
	rate pgtype.Numeric,
) exchangeRateResponse {
	return exchangeRateResponse{
		ID:   id,
		From: from,
		To:   to,
		Rate: numericToString(rate),
	}
}

// Convert PostgreSQL NUMERIC to a decimal string without float64 precision loss.
func numericToString(n pgtype.Numeric) string {
	if !n.Valid || n.Int == nil {
		return ""
	}

	if n.NaN {
		return "NaN"
	}

	value := n.Int.String()

	if n.Exp >= 0 {
		return value + strings.Repeat("0", int(n.Exp))
	}

	negative := strings.HasPrefix(value, "-")
	digits := strings.TrimPrefix(value, "-")
	places := int(-n.Exp)

	if places >= len(digits) {
		digits = strings.Repeat("0", places-len(digits)+1) + digits
	}

	point := len(digits) - places
	result := digits[:point] + "." + digits[point:]
	result = strings.TrimRight(result, "0")
	result = strings.TrimRight(result, ".")

	if negative {
		result = "-" + result
	}

	return result
}

type exchangeRatePairQuery struct {
	From string `form:"from" binding:"required,len=3"`
	To   string `form:"to" binding:"required,len=3"`
}

// GET /exchange-rates?from=USD&to=EUR
func (server *Server) GetExchangeRate(ctx *gin.Context) {
	var req exchangeRatePairQuery
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	from := strings.ToUpper(req.From)
	to := strings.ToUpper(req.To)

	if from == to {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "from and to currencies must be different",
		})
		return
	}

	rate, err := server.store.GetLatestExchangeRate(
		ctx,
		db.GetLatestExchangeRateParams{
			BaseCurrency:  from,
			QuoteCurrency: to,
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newExchangeRateResponse(
		rate.ID,
		rate.BaseCurrency,
		rate.QuoteCurrency,
		rate.Rate,
	))
}

// GET /exchange-rates/all
func (server *Server) ListLatestExchangeRates(ctx *gin.Context) {
	rows, err := server.store.ListLatestExchangeRates(ctx)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rates := make([]exchangeRateResponse, 0, len(rows))

	for _, r := range rows {
		rates = append(rates, newExchangeRateResponse(
			r.ID,
			r.BaseCurrency,
			r.QuoteCurrency,
			r.Rate,
		))
	}

	ctx.JSON(http.StatusOK, rates)
}

type exchangeRateHistoryQuery struct {
	From string `form:"from" binding:"required,len=3"`
	To   string `form:"to" binding:"required,len=3"`
}

// GET /exchange-rates/history?from=USD&to=EUR
func (server *Server) GetExchangeRateHistory(ctx *gin.Context) {
	var req exchangeRateHistoryQuery
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	from := strings.ToUpper(req.From)
	to := strings.ToUpper(req.To)

	if from == to {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "from and to currencies must be different",
		})
		return
	}

	rows, err := server.store.ListExchangeRateHistory(
		ctx,
		db.ListExchangeRateHistoryParams{
			BaseCurrency:  from,
			QuoteCurrency: to,
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rates := make([]exchangeRateResponse, 0, len(rows))

	for _, r := range rows {
		rates = append(rates, newExchangeRateResponse(
			r.ID,
			r.BaseCurrency,
			r.QuoteCurrency,
			r.Rate,
		))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"from":  from,
		"to":    to,
		"rates": rates,
	})
}

type adminListExchangeRatesRequest struct {
	From     *string `form:"from" binding:"omitempty,len=3"`
	To       *string `form:"to" binding:"omitempty,len=3"`
	PageID   int32   `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32   `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GET /admin/exchange-rates
func (server *Server) AdminListExchangeRates(ctx *gin.Context) {
	var req adminListExchangeRatesRequest
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

	arg := db.ListExchangeRatesAdminParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}

	if req.From != nil {
		arg.BaseCurrency = pgtype.Text{
			String: strings.ToUpper(*req.From),
			Valid:  true,
		}
	}

	if req.To != nil {
		arg.QuoteCurrency = pgtype.Text{
			String: strings.ToUpper(*req.To),
			Valid:  true,
		}
	}

	rows, err := server.store.ListExchangeRatesAdmin(ctx, arg)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rates := make([]exchangeRateResponse, 0, len(rows))

	for _, r := range rows {
		rates = append(rates, newExchangeRateResponse(
			r.ID,
			r.BaseCurrency,
			r.QuoteCurrency,
			r.Rate,
		))
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":   req.PageID,
		"page_size": req.PageSize,
		"rates":     rates,
	})
}
