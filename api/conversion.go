package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const conversionQuoteTTL = 30 * time.Second

var errInvalidQuote = errors.New("invalid quote")

type conversionQuote struct {
	UserID        int64   `json:"u"`
	FromAccountID int64   `json:"f"`
	ToAccountID   int64   `json:"t"`
	FromAmount    int64   `json:"a"`
	Rate          float64 `json:"r"`
	Fee           int64   `json:"fee"`
	ExpiresAt     int64   `json:"e"`
	Nonce         string  `json:"n"`
}

func (server *Server) quoteMAC(body string) string {
	mac := hmac.New(sha256.New, []byte(server.config.TokenSymmetricKey))
	mac.Write([]byte("conversion-quote:" + body))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (server *Server) signQuote(q conversionQuote) (string, error) {
	payload, err := json.Marshal(q)
	if err != nil {
		return "", err
	}
	body := base64.RawURLEncoding.EncodeToString(payload)
	return body + "." + server.quoteMAC(body), nil
}

func (server *Server) verifyQuote(token string) (conversionQuote, error) {
	var q conversionQuote

	body, sig, ok := strings.Cut(token, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(server.quoteMAC(body))) {
		return q, errInvalidQuote
	}

	payload, err := base64.RawURLEncoding.DecodeString(body)
	if err != nil {
		return q, errInvalidQuote
	}
	if err := json.Unmarshal(payload, &q); err != nil {
		return q, errInvalidQuote
	}

	return q, nil
}

func numericToFloat(n pgtype.Numeric) (float64, bool) {
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0, false
	}
	return f.Float64, true
}

func (server *Server) latestRate(ctx *gin.Context, base, quote string) (float64, bool) {
	row, err := server.store.GetExchangeRate(ctx, db.GetExchangeRateParams{
		BaseCurrency:  base,
		QuoteCurrency: quote,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "No exchange rate available for this currency pair"})
			return 0, false
		}
		handleDBError(ctx, err)
		return 0, false
	}

	rate, ok := numericToFloat(row.Rate)
	if !ok || rate <= 0 {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid exchange rate"})
		return 0, false
	}
	return rate, true
}

func (server *Server) conversionFee(ctx *gin.Context, currency string, amount int64) (int64, error) {
	rule, err := server.store.GetActiveFeeRule(ctx, db.GetActiveFeeRuleParams{
		OperationType: "conversion",
		Currency:      currency,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return store.CalculateFee(rule, amount), nil
}

func handleConversionError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrSameAccountConversion),
		errors.Is(err, store.ErrSameCurrencyConversion):
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
	case errors.Is(err, store.ErrCrossUserConversion):
		ctx.JSON(http.StatusForbidden, errorResponse(err))
	case errors.Is(err, store.ErrInsufficientFunds):
		ctx.JSON(http.StatusUnprocessableEntity, errorResponse(err))
	case errors.Is(err, store.ErrAccountNotActive):
		ctx.JSON(http.StatusConflict, errorResponse(err))
	case errors.Is(err, store.ErrAccountNotFound),
		errors.Is(err, pgx.ErrNoRows):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
	default:
		handleDBError(ctx, err)
	}
}

type conversionResponse struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	FromAccountID int64     `json:"from_account_id"`
	ToAccountID   int64     `json:"to_account_id"`
	FromAmount    int64     `json:"from_amount"`
	ToAmount      int64     `json:"to_amount"`
	FromCurrency  string    `json:"from_currency"`
	ToCurrency    string    `json:"to_currency"`
	ExchangeRate  float64   `json:"exchange_rate"`
	Fee           int64     `json:"fee"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func newConversionResponse(c db.Conversion) conversionResponse {
	rate, _ := numericToFloat(c.ExchangeRate)
	return conversionResponse{
		ID:            c.ID,
		UserID:        c.UserID,
		FromAccountID: c.FromAccountID,
		ToAccountID:   c.ToAccountID,
		FromAmount:    c.FromAmount,
		ToAmount:      c.ToAmount,
		FromCurrency:  c.FromCurrency,
		ToCurrency:    c.ToCurrency,
		ExchangeRate:  rate,
		Fee:           c.Fee,
		Status:        c.Status,
		CreatedAt:     c.CreatedAt.Time,
	}
}

func newConversionResponses(rows []db.Conversion) []conversionResponse {
	conversions := make([]conversionResponse, 0, len(rows))
	for _, row := range rows {
		conversions = append(conversions, newConversionResponse(row))
	}
	return conversions
}

type conversionQuoteRequest struct {
	FromAccountID int64 `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64 `json:"to_account_id" binding:"required,min=1"`
	FromAmount    int64 `json:"from_amount" binding:"required,gt=0"`
}

// POST /conversions/quote
func (server *Server) CreateConversionQuote(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req conversionQuoteRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.FromAccountID == req.ToAccountID {
		ctx.JSON(http.StatusBadRequest, errorResponse(store.ErrSameAccountConversion))
		return
	}

	fromAccount, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     req.FromAccountID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	toAccount, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     req.ToAccountID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if fromAccount.Currency == toAccount.Currency {
		ctx.JSON(http.StatusBadRequest, errorResponse(store.ErrSameCurrencyConversion))
		return
	}

	rate, ok := server.latestRate(ctx, fromAccount.Currency, toAccount.Currency)
	if !ok {
		return
	}

	toAmount := int64(math.Round(float64(req.FromAmount) * rate))
	if toAmount <= 0 {
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Amount is too small to convert"})
		return
	}

	fee, err := server.conversionFee(ctx, fromAccount.Currency, req.FromAmount)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	totalDebit := req.FromAmount + fee
	if fromAccount.Balance < totalDebit {
		ctx.JSON(http.StatusUnprocessableEntity, errorResponse(store.ErrInsufficientFunds))
		return
	}

	nonce, err := newIdempotencyKey()
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create quote"})
		return
	}

	expiresAt := time.Now().Add(conversionQuoteTTL)
	token, err := server.signQuote(conversionQuote{
		UserID:        payload.UserID,
		FromAccountID: fromAccount.ID,
		ToAccountID:   toAccount.ID,
		FromAmount:    req.FromAmount,
		Rate:          rate,
		Fee:           fee,
		ExpiresAt:     expiresAt.Unix(),
		Nonce:         nonce,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create quote"})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"quote_id":        token,
		"from_account_id": fromAccount.ID,
		"to_account_id":   toAccount.ID,
		"from_currency":   fromAccount.Currency,
		"to_currency":     toAccount.Currency,
		"from_amount":     req.FromAmount,
		"rate":            rate,
		"fee":             fee,
		"total_debit":     totalDebit,
		"to_amount":       toAmount,
		"expires_at":      expiresAt.UTC(),
	})
}

type createConversionRequest struct {
	QuoteID string `json:"quote_id" binding:"required"`
}

// POST /conversions
func (server *Server) CreateConversion(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req createConversionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	quote, err := server.verifyQuote(req.QuoteID)
	if err != nil || quote.UserID != payload.UserID {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quote"})
		return
	}

	idempotencyKey := pgtype.Text{
		String: fmt.Sprintf("%d:conv:%s", payload.UserID, quote.Nonce),
		Valid:  true,
	}

	existing, err := server.store.GetConversionByIdempotencyKey(ctx, idempotencyKey)
	if err == nil {
		ctx.JSON(http.StatusOK, newConversionResponse(existing))
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		handleDBError(ctx, err)
		return
	}

	if time.Now().Unix() > quote.ExpiresAt {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Quote expired, request a new one"})
		return
	}

	fromAccount, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     quote.FromAccountID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	toAccount, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     quote.ToAccountID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rate, ok := server.latestRate(ctx, fromAccount.Currency, toAccount.Currency)
	if !ok {
		return
	}
	if rate != quote.Rate {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Exchange rate changed, request a new quote"})
		return
	}

	result, err := server.store.CreateConversionTransaction(ctx, store.CreateConversionTransactionArgs{
		UserID:         payload.UserID,
		FromAccountID:  quote.FromAccountID,
		ToAccountID:    quote.ToAccountID,
		FromAmount:     quote.FromAmount,
		Fee:            quote.Fee,
		Status:         "completed",
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		handleConversionError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newConversionResponse(result.Conversion))
}

type listConversionsRequest struct {
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GET /conversions
func (server *Server) ListConversions(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req listConversionsRequest
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

	rows, err := server.store.ListConversionsByUserID(ctx, db.ListConversionsByUserIDParams{
		UserID: payload.UserID,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":     req.PageID,
		"page_size":   req.PageSize,
		"conversions": newConversionResponses(rows),
	})
}

type conversionURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GET /conversions/:id
func (server *Server) GetConversion(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri conversionURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	conversion, err := server.store.GetConversionByID(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if conversion.UserID != payload.UserID {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Conversion not found"})
		return
	}

	ctx.JSON(http.StatusOK, newConversionResponse(conversion))
}

type adminListConversionsRequest struct {
	UserID       *int64  `form:"user_id" binding:"omitempty,min=1"`
	FromCurrency *string `form:"from_currency" binding:"omitempty,len=3"`
	ToCurrency   *string `form:"to_currency" binding:"omitempty,len=3"`
	From         *string `form:"from" binding:"omitempty,datetime=2006-01-02"`
	To           *string `form:"to" binding:"omitempty,datetime=2006-01-02"`
	PageID       int32   `form:"page_id" binding:"omitempty,min=1"`
	PageSize     int32   `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GET /admin/conversions
func (server *Server) AdminListConversions(ctx *gin.Context) {
	var req adminListConversionsRequest
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

	arg := db.ListConversionsAdminParams{
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	}
	if req.UserID != nil {
		arg.UserID = pgtype.Int8{Int64: *req.UserID, Valid: true}
	}
	if req.FromCurrency != nil {
		arg.FromCurrency = pgtype.Text{String: strings.ToUpper(*req.FromCurrency), Valid: true}
	}
	if req.ToCurrency != nil {
		arg.ToCurrency = pgtype.Text{String: strings.ToUpper(*req.ToCurrency), Valid: true}
	}
	if req.From != nil {
		from, err := time.Parse("2006-01-02", *req.From)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "from must be YYYY-MM-DD"})
			return
		}
		arg.FromDate = pgtype.Timestamptz{Time: from, Valid: true}
	}
	if req.To != nil {
		to, err := time.Parse("2006-01-02", *req.To)
		if err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "to must be YYYY-MM-DD"})
			return
		}
		arg.ToDate = pgtype.Timestamptz{Time: to.AddDate(0, 0, 1), Valid: true}
	}
	if arg.FromDate.Valid && arg.ToDate.Valid && !arg.FromDate.Time.Before(arg.ToDate.Time) {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "from must not be after to"})
		return
	}

	rows, err := server.store.ListConversionsAdmin(ctx, arg)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":     req.PageID,
		"page_size":   req.PageSize,
		"conversions": newConversionResponses(rows),
	})
}

// GET /admin/conversions/:id
func (server *Server) AdminGetConversion(ctx *gin.Context) {
	var uri conversionURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	conversion, err := server.store.GetConversionByID(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, newConversionResponse(conversion))
}

// POST /admin/conversions/:id/reverse
func (server *Server) AdminReverseConversion(ctx *gin.Context) {
	var uri conversionURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req adminActionRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	reversed, err := server.store.ReverseConversionTransaction(ctx, store.ReverseConversionTransactionArgs{
		ConversionID: uri.ID,
		Reason:       req.Reason,
	})
	if err != nil {
		switch {
		case errors.Is(err, store.ErrConversionNotReversible):
			ctx.JSON(http.StatusConflict, errorResponse(err))
		case errors.Is(err, store.ErrInsufficientFunds):
			ctx.JSON(http.StatusUnprocessableEntity, gin.H{
				"error": "Destination account balance is too low to reverse this conversion",
			})
		case errors.Is(err, pgx.ErrNoRows):
			ctx.JSON(http.StatusNotFound, gin.H{"error": "Conversion not found"})
		default:
			handleDBError(ctx, err)
		}
		return
	}

	ctx.JSON(http.StatusOK, newConversionResponse(reversed))
}
