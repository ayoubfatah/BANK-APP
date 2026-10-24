package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/db/store"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var ErrCardLimitReached = errors.New("card limit per user reached")

type cardResponse struct {
	LastFour string `json:"last_four"`
	Brand    string `json:"brand"`

	Status     string    `json:"status"`
	ExpiryDate time.Time `json:"expiry_date"`
}

func newCardResponse(card db.Card) cardResponse {
	return cardResponse{
		LastFour:   card.LastFour,
		Brand:      card.Brand,
		Status:     card.Status,
		ExpiryDate: card.ExpiryDate.Time,
	}
}

func (server *Server) cardTransitionError(ctx *gin.Context, err error, id, userID int64) {
	if errors.Is(err, pgx.ErrNoRows) {
		card, getErr := server.store.GetCardByIDAndUserID(ctx, db.GetCardByIDAndUserIDParams{ID: id, UserID: userID})
		if getErr == nil {
			ctx.JSON(http.StatusConflict, gin.H{"error": "card is " + card.Status})
			return
		}
		err = getErr
	}
	handleDBError(ctx, err)
}

type CreateCardRequest struct {
	AccountID int64  `json:"account_id" binding:"required min=1"`
	Brand     string `json:"brand" binding:"required,oneof=visa mastercard amex discover cmi"`
	CardType  string `json:"card_type" binding:"required,oneof =physical virtual"`
}

func (server *Server) createCard(ctx *gin.Context) {
	cardNumber := utils.RandomNumbers(14)
	lastFour := cardNumber[len(cardNumber)-4:]
	randomCvv := utils.RandomNumbers(3)
	payload := getAuthPayload(ctx)

	var req CreateCardRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	account, err := server.store.GetAccountByIDAndUserID(ctx, db.GetAccountByIDAndUserIDParams{
		ID:     req.AccountID,
		UserID: payload.UserID,
	})

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if account.Status != "active" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "account is not active",
		})
		return
	}

	card, _, err := server.store.CreateCardTx(ctx, store.CreateCardTxParams{
		UserID:     account.UserID,
		AccountID:  account.ID,
		Currency:   account.Currency,
		CardNumber: cardNumber,
		LastFour:   lastFour,
		Cvv:        randomCvv,
		Brand:      req.Brand,
		CardType:   req.CardType,
		Status:     "active",
		ExpiryDate: pgtype.Date{
			Time:  time.Now().AddDate(5, 0, 0),
			Valid: true,
		},
	})

	if errors.Is(err, ErrCardLimitReached) {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "maximum of 4 cards reached",
		})
		return
	}

	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newCardResponse(card))

}

type GetCardParam struct {
	ID int64 `uri:"id" json:"id" biding:"required,min=1"`
}

func (server *Server) getCard(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var param GetCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// check if card belongs to the user
	card, err := server.store.GetCard(ctx, param.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if card.UserID != payload.UserID {
		ctx.JSON(http.StatusForbidden, errorResponse(errors.New("This card doesn't belong to you 🤨📸")))
		return
	}
	ctx.JSON(http.StatusCreated, newCardResponse(card))
}

func (server *Server) getCards(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	// check if card belongs to the user
	cards, err := server.store.ListCardsByUserID(ctx, payload.UserID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	newCardsResponse := make([]cardResponse, 0, len(cards))

	for _, card := range cards {
		newCardsResponse = append(newCardsResponse, newCardResponse(card))
	}

	ctx.JSON(http.StatusOK, newCardsResponse)
}

// FREEZE CARD

type FreezeCardParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) freezeCard(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var param FreezeCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	freezedCard, err := server.store.FreezeCard(ctx, db.FreezeCardParams{
		ID:     param.ID,
		UserID: payload.UserID,
	})

	if err != nil {
		server.cardTransitionError(ctx, err, param.ID, payload.UserID)
		return
	}
	ctx.JSON(http.StatusOK, newCardResponse(freezedCard))
}

// Unfreeze card

type UnFreezeCardParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

func (server *Server) UnfreezeCard(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	var param UnFreezeCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	card, err := server.store.UnFreezeCard(ctx, db.UnFreezeCardParams{
		ID:     param.ID,
		UserID: payload.UserID,
	})

	if err != nil {
		server.cardTransitionError(ctx, err, param.ID, payload.UserID)
		return
	}
	ctx.JSON(http.StatusOK, newCardResponse(card))
}

type CancelCardParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type CancelCardReq struct {
	Reason *string `json:"reason" binding:"omitempty"`
}

func (server *Server) cancelCard(ctx *gin.Context) {

	payload := getAuthPayload(ctx)
	var param CancelCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req CancelCardReq
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	canceledCard, err := server.store.CancelCard(ctx, db.CancelCardParams{
		ID:     param.ID,
		UserID: payload.UserID,
	})

	if err != nil {
		server.cardTransitionError(ctx, err, param.ID, payload.UserID)
		return
	}
	ctx.JSON(http.StatusOK, newCardResponse(canceledCard))
}

//  /CARD/:ID/LIMIT

type cardLimitsResponse struct {
	CardID       int64 `json:"card_id"`
	DailyLimit   int64 `json:"daily_limit"`
	MonthlyLimit int64 `json:"monthly_limit"`
}

type SetCardLimitsParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type setCardLimitsRequest struct {
	DailyLimit   int64 `json:"daily_limit" binding:"omitempty,gt=0"`
	MonthlyLimit int64 `json:"monthly_limit" binding:"omitempty,gt=0"`
}

func (server *Server) changeSpendLimit(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var param SetCardLimitsParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	var req setCardLimitsRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if req.DailyLimit > req.MonthlyLimit {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "daily_limit cannot be greater than monthly_limit"})
		return
	}

	card, err := server.store.GetCardByIDAndUserID(ctx, db.GetCardByIDAndUserIDParams{
		ID:     param.ID,
		UserID: payload.UserID,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}
	if card.Status != "active" && card.Status != "frozen" {
		ctx.JSON(http.StatusConflict, gin.H{"error": "card is " + card.Status})
		return
	}

	var daily, monthly pgtype.Int8

	daily = pgtype.Int8{Int64: req.DailyLimit, Valid: true}

	monthly = pgtype.Int8{Int64: req.MonthlyLimit, Valid: true}

	limits, err := server.store.UpdateCardSpendingLimits(ctx, db.UpdateCardSpendingLimitsParams{
		CardID:       card.ID,
		UserID:       payload.UserID,
		DailyLimit:   daily,
		MonthlyLimit: monthly,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		// cards created before limits existed have no row yet
		limits, err = server.store.CreateCardSpendingLimits(ctx, db.CreateCardSpendingLimitsParams{
			CardID:       card.ID,
			DailyLimit:   daily,
			MonthlyLimit: monthly,
		})
	}
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, cardLimitsResponse{
		CardID:       limits.CardID,
		DailyLimit:   limits.DailyLimit.Int64,
		MonthlyLimit: limits.MonthlyLimit.Int64,
	})
}

// replace

type ReplaceCardParam struct {
	ID int64 `uri:"id" binding:"required,min=1"`
}

type replaceCardRequest struct {
	Reason string `json:"reason" binding:"required,oneof=lost stolen damaged"`
}

// POST /cards/:id/replace
func (server *Server) ReplaceCard(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	var param ReplaceCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	var req replaceCardRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	oldCard, err := server.store.GetCardByIDAndUserID(
		ctx,
		db.GetCardByIDAndUserIDParams{
			ID:     param.ID,
			UserID: payload.UserID,
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if oldCard.Status != "active" && oldCard.Status != "frozen" {
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "card is " + oldCard.Status,
		})
		return
	}

	account, err := server.store.GetAccountByIDAndUserID(
		ctx,
		db.GetAccountByIDAndUserIDParams{
			ID:     oldCard.AccountID,
			UserID: payload.UserID,
		},
	)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	if account.Status != "active" {
		ctx.JSON(http.StatusConflict, gin.H{
			"error": "account is " + account.Status,
		})
		return
	}

	cardNumber := utils.RandomNumbers(14)
	lastFour := cardNumber[len(cardNumber)-4:]
	randomCvv := utils.RandomNumbers(3)

	newCard := db.CreateCardParams{
		UserID:     oldCard.UserID,
		AccountID:  oldCard.AccountID,
		CardNumber: cardNumber,
		LastFour:   lastFour,
		Cvv:        randomCvv,
		Brand:      oldCard.Brand,
		CardType:   oldCard.CardType,
		Status:     "active",
		ExpiryDate: pgtype.Date{
			Time:  time.Now().AddDate(5, 0, 0),
			Valid: true,
		},
	}

	result, err := server.store.ReplaceCardTx(ctx, store.ReplaceCardTxArgs{
		OldCardID: param.ID,
		UserID:    payload.UserID,
		Reason:    req.Reason,
		Currency:  account.Currency,
		NewCard:   newCard,
	})
	if err != nil {
		server.cardTransitionError(ctx, err, param.ID, payload.UserID)
		return
	}

	ctx.JSON(http.StatusCreated, gin.H{
		"old_card": newCardResponse(result.OldCard),
		"new_card": newCardResponse(result.NewCard),
	})
}

// ADMIN LIST CARDS

type adminListCardsRequest struct {
	LastFour *string `form:"last_four"`
	Brand    *string `form:"brand"`
	Status   *string `form:"status" binding:"omitempty,oneof=active blocked frozen expired cancelled"`
	CardType *string `form:"card_type" binding:"omitempty,oneof=virtual physical"`
	UserID   *int64  `form:"user_id" binding:"omitempty,gt=0"`
	PageID   int32   `form:"page_id,default=1" binding:"gte=1"`
	PageSize int32   `form:"page_size,default=10" binding:"gte=1,lte=100"`
}

func (server *Server) AdminListCards(ctx *gin.Context) {
	payload := getAuthPayload(ctx)
	if payload.Role != "admin" {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("You are not allowed to perform this action")))
		return
	}
	var req adminListCardsRequest

	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	offset := (req.PageID - 1) * req.PageSize

	var lastFour, brand, status, cardType pgtype.Text
	var userID pgtype.Int8

	if req.LastFour != nil {
		lastFour = pgtype.Text{String: *req.LastFour, Valid: true}
	}

	if req.Brand != nil {
		brand = pgtype.Text{String: *req.Brand, Valid: true}
	}

	if req.Status != nil {
		status = pgtype.Text{String: *req.Status, Valid: true}
	}

	if req.CardType != nil {
		cardType = pgtype.Text{String: *req.CardType, Valid: true}
	}

	if req.UserID != nil {
		userID = pgtype.Int8{Int64: *req.UserID, Valid: true}
	}

	cards, err := server.store.AdminListCards(ctx, db.AdminListCardsParams{
		LastFour: lastFour,
		Brand:    brand,
		Status:   status,
		CardType: cardType,
		UserID:   userID,
		Offset:   offset,
		PageSize: req.PageSize,
	})
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data":      cards,
		"page_id":   req.PageID,
		"page_size": req.PageSize,
	})

}

// admin get card

type AdminGetCardParam struct {
	ID int64 `uri:"id" json:"id" biding:"required,min=1"`
}

func (server *Server) AdminGetCard(ctx *gin.Context) {
	payload := getAuthPayload(ctx)

	if payload.Role != "admin" {
		ctx.JSON(http.StatusUnauthorized, errorResponse(errors.New("You are not allowed to perform this action")))
		return
	}
	var param AdminGetCardParam
	if err := ctx.ShouldBindUri(&param); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	// check if card belongs to the user
	card, err := server.store.GetCard(ctx, param.ID)
	if err != nil {
		handleDBError(ctx, err)
		return
	}

	ctx.JSON(http.StatusCreated, newCardResponse(card))
}
