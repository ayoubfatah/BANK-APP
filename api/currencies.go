package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	db "github.com/bank-app/db/sqlc"
)

type currencyResponse struct {
	Code     string `json:"code"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals int32  `json:"decimals"` // 2 for USD (cents), 0 for JPY, 3 for KWD
	IsActive bool   `json:"is_active"`
}

func newCurrencyResponse(c db.Currency) currencyResponse {
	return currencyResponse{
		Code:     c.Code,
		Name:     c.Name,
		Symbol:   c.Symbol,
		Decimals: c.Decimals,
		IsActive: c.IsActive,
	}
}

func newCurrencyList(rows []db.Currency) []currencyResponse {
	out := make([]currencyResponse, 0, len(rows)) // [] not null
	for _, c := range rows {
		out = append(out, newCurrencyResponse(c))
	}
	return out
}

func (server *Server) getCurrencyByCode(ctx *gin.Context) {
	code := strings.ToUpper(ctx.Param("code"))

	if len(code) != 3 || !isLettersOnly(code) {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "currency code must be 3 letters",
		})
		return
	}

	currency, err := server.store.GetCurrency(ctx, code)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newCurrencyResponse(currency))
}
func isLettersOnly(s string) bool {
	for _, r := range s {
		if r < 'A' || r > 'Z' {
			return false
		}
	}
	return true
}

// ================= GET /currencies (public) =================

func (server *Server) listActiveCurrencies(ctx *gin.Context) {
	rows, err := server.store.ListActiveCurrencies(ctx)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": newCurrencyList(rows)})
}

// ================= GET /admin/currencies (optional: admins also need to see inactive ones) =================

type listAllCurrenciesQuery struct {
	Limit int `form:"limit,default=10" binding:"gte=1,max=100"`
	Page  int `form:"page,default=1" binding:"gte=1"`
}

func (server *Server) listAllCurrencies(ctx *gin.Context) {

	var queries listAllCurrenciesQuery

	err := ctx.ShouldBindQuery(&queries)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	offset := (queries.Page - 1) * queries.Limit
	rows, err := server.store.ListCurrencies(ctx, db.ListCurrenciesParams{
		Limit:  int32(queries.Limit),
		Offset: int32(offset),
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"data": newCurrencyList(rows)})
}

type createCurrencyRequest struct {
	Code     string `json:"code" binding:"required,len=3,alpha"`
	Name     string `json:"name" binding:"required,max=100"`
	Symbol   string `json:"symbol" binding:"required,max=10"`
	Decimals *int32 `json:"decimals" binding:"required,min=0,max=8"`
}

func (server *Server) createCurrency(ctx *gin.Context) {
	var req createCurrencyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	currency, err := server.store.CreateCurrency(ctx, db.CreateCurrencyParams{
		Code:     strings.ToUpper(req.Code),
		Name:     req.Name,
		Symbol:   req.Symbol,
		Decimals: *req.Decimals,
	})
	if err != nil {
		handleDBError(ctx, err) // duplicate code -> 409
		return
	}

	ctx.JSON(http.StatusCreated, newCurrencyResponse(currency))
}

// ADMIN

type setCurrencyActiveRequest struct {
	Code     string `uri:"code" binding:"required"`
	IsActive *bool  `json:"is_active" binding:"required"`
}

func (server *Server) setCurrencyActive(ctx *gin.Context) {
	var params setCurrencyActiveRequest

	if err := ctx.ShouldBindUri(&params); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	_, err := server.store.GetCurrency(ctx, params.Code)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(errors.New("This  currency doesn't exist")))

	}

	var req setCurrencyActiveRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	currency, err := server.store.UpdateCurrencyActiveState(ctx, db.UpdateCurrencyActiveStateParams{
		Code:     req.Code,
		IsActive: *req.IsActive,
	})
	if err != nil {
		handleDBError(ctx, err) // unknown code -> 404
		return
	}

	ctx.JSON(http.StatusOK, newCurrencyResponse(currency))
}
