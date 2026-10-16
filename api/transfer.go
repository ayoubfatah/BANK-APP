package api

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type transferResponse struct {
	ID            int64      `json:"id"`
	FromAccountID int64      `json:"from_account_id"`
	ToAccountID   int64      `json:"to_account_id"`
	Amount        int64      `json:"amount"`
	Currency      string     `json:"currency"`
	Status        string     `json:"status"`
	Reference     string     `json:"reference,omitempty"`
	FailureReason string     `json:"failure_reason,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
}

func newTransferResponse(t db.Transfer) transferResponse {
	res := transferResponse{
		ID:            t.ID,
		FromAccountID: t.FromAccountID.Int64,
		ToAccountID:   t.ToAccountID.Int64,
		Amount:        t.Amount,
		Currency:      t.Currency,
		Status:        t.Status,
		Reference:     t.Reference.String,
		FailureReason: t.FailureReason.String,
		CreatedAt:     t.CreatedAt.Time,
	}
	if t.CompletedAt.Valid {
		completedAt := t.CompletedAt.Time
		res.CompletedAt = &completedAt
	}
	return res
}

func newTransferResponses(rows []db.Transfer) []transferResponse {
	transfers := make([]transferResponse, 0, len(rows))
	for _, row := range rows {
		transfers = append(transfers, newTransferResponse(row))
	}
	return transfers
}

func handleTransferError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, store.ErrSameAccount):
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
	case errors.Is(err, store.ErrCurrencyMismatch),
		errors.Is(err, store.ErrInsufficientFunds):
		ctx.JSON(http.StatusUnprocessableEntity, errorResponse(err))
	case errors.Is(err, store.ErrAccountNotFound),
		errors.Is(err, pgx.ErrNoRows):
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Account not found"})
	default:
		handleDBError(ctx, err)
	}
}

func newIdempotencyKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (server *Server) userOwnsTransfer(
	ctx *gin.Context,
	userID int64,
	transfer db.Transfer,
) (bool, error) {
	for _, accountID := range []pgtype.Int8{transfer.FromAccountID, transfer.ToAccountID} {
		if !accountID.Valid {
			continue
		}
		_, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
			ID:     accountID.Int64,
			UserID: userID,
		})
		if err == nil {
			return true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return false, err
		}
	}
	return false, nil
}

type createTransferRequest struct {
	FromAccountID int64  `json:"from_account_id" binding:"required,min=1"`
	ToAccountID   int64  `json:"to_account_id" binding:"required,min=1"`
	Amount        int64  `json:"amount" binding:"required,gt=0"`
	Reference     string `json:"reference" binding:"omitempty,max=140"`
}

// POST /transfers
func (server *Server) CreateTransfer(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req createTransferRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	key := ctx.GetHeader("Idempotency-Key")
	if key == "" {
		generated, err := newIdempotencyKey()
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Could not create transfer"})
			return
		}
		key = generated
	}
	if len(key) > 64 {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Idempotency-Key must be at most 64 characters"})
		return
	}
	idempotencyKey := pgtype.Text{
		String: fmt.Sprintf("%d:%s", payload.UserID, key),
		Valid:  true,
	}

	fromAccount, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     req.FromAccountID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if fromAccount.Status != "active" {
		ctx.JSON(http.StatusConflict, gin.H{"error": "Source account is not active"})
		return
	}

	err = server.store.CreateTransferTransaction(ctx, store.CreateTransferTransactionArgs{
		FromAccountID:  req.FromAccountID,
		ToAccountID:    req.ToAccountID,
		Amount:         req.Amount,
		Currency:       fromAccount.Currency,
		Status:         "pending",
		Reference:      pgtype.Text{String: req.Reference, Valid: req.Reference != ""},
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		handleTransferError(ctx, err)
		return
	}

	transfer, err := server.store.GetTransferByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if transfer.FromAccountID.Int64 != req.FromAccountID ||
		transfer.ToAccountID.Int64 != req.ToAccountID ||
		transfer.Amount != req.Amount {
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "Idempotency-Key was already used with different transfer details",
		})
		return
	}

	ctx.JSON(http.StatusCreated, newTransferResponse(transfer))
}

type listTransfersRequest struct {
	PageID   int32 `form:"page_id" binding:"omitempty,min=1"`
	PageSize int32 `form:"page_size" binding:"omitempty,min=1,max=100"`
}

// GET /transfers
func (server *Server) ListTransfers(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var req listTransfersRequest
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

	rows, err := server.store.ListTransfersByUserID(ctx, db.ListTransfersByUserIDParams{
		UserID: payload.UserID,
		Limit:  req.PageSize,
		Offset: (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":   req.PageID,
		"page_size": req.PageSize,
		"transfers": newTransferResponses(rows),
	})
}

type transferURI struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

// GET /transfers/:id
func (server *Server) GetTransfer(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri transferURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	transfer, err := server.store.GetTransfer(ctx, uri.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	owns, err := server.userOwnsTransfer(ctx, payload.UserID, transfer)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if !owns {
		ctx.JSON(http.StatusNotFound, gin.H{"error": "Transfer not found"})
		return
	}

	ctx.JSON(http.StatusOK, newTransferResponse(transfer))
}

// GET /accounts/:id/transfers
func (server *Server) ListAccountTransfers(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var uri transferURI
	if err := ctx.ShouldBindUri(&uri); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req listTransfersRequest
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

	account, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     uri.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	rows, err := server.store.ListTransfersByAccountID(ctx, db.ListTransfersByAccountIDParams{
		FromAccountID: pgtype.Int8{Int64: account.ID, Valid: true},
		Limit:         req.PageSize,
		Offset:        (req.PageID - 1) * req.PageSize,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"page_id":   req.PageID,
		"page_size": req.PageSize,
		"transfers": newTransferResponses(rows),
	})
}
