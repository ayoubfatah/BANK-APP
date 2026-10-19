package api

import (
	"errors"
	"net/http"
	"time"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
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

	card, err := server.store.CreateCardTx(ctx, db.CreateCardParams{
		UserID:     account.UserID,
		AccountID:  account.ID,
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
