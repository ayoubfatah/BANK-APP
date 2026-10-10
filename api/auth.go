package api

import (
	"errors"
	"net/http"
	"net/netip"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	db "github.com/bank-app/db/sqlc"
	"github.com/bank-app/utils"
)

type loginUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type loginUserResponse struct {
	SessionID  uuid.UUID `json:"session_id"`
	AccessToken          string      `json:"access_token"`
	RefreshToken          string      `json:"refresh_token"`
	AccessTokenExpiresAt time.Time   `json:"access_token_expires_at"`
	RefreshTokenExpiresAt time.Time   `json:"refresh_token_expires_at"`
	User                 userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	invalid := errors.New("invalid credentials")

	user, err := server.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusUnauthorized, errorResponse(invalid))
		return
	}

	handleDBError(ctx, err)
	return
	}

	if !utils.CheckPasswordHash(req.Password, user.PasswordHash) {
		ctx.JSON(http.StatusUnauthorized, errorResponse(invalid))
		return
	}

	if user.Status != "active" {
		ctx.JSON(http.StatusForbidden, gin.H{
			"error": "account is not active",
		})
		return
	}

	accessToken , accessPayload , err := server.tokenMaker.CreateToken(
		user.Email,
		user.ID,
		user.Role,
		server.config.AccessTokenDuration,
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	refreshToken, refreshPayload , err := server.tokenMaker.CreateToken(
		user.Email,
		user.ID,
		user.Role,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	userAgent := ctx.GetHeader("User-Agent")
	clientIP := ctx.ClientIP()
	
	_,  err=  server.store.CreateRefreshToken(ctx ,  db.CreateRefreshTokenParams{
	ID  : pgtype.UUID{
			Bytes: refreshPayload.ID,
			Valid: true,
	},
    UserID : user.ID ,
    TokenHash: refreshToken ,
    UserAgent:  userAgent,
	ClientIp: netip.MustParseAddr(clientIP),
    ExpiresAt: pgtype.Timestamptz{
		Time:  refreshPayload.ExpiredAt,
		Valid: true,
	} ,
	
	 })

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	ctx.JSON(http.StatusOK, loginUserResponse{
		User:                 newUserResponse(user),
		AccessToken: accessToken,
		AccessTokenExpiresAt:accessPayload.ExpiredAt ,
		RefreshToken: refreshToken,
		RefreshTokenExpiresAt :refreshPayload.ExpiredAt , 

	})
}