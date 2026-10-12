package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

type renewAccessTokenRequest struct {
	RefreshToken    string `json:"refresh_token" binding:"required"`
}

type renewAccessTokenResponse struct {
	AccessToken          string      `json:"access_token"`
	AccessTokenExpiresAt time.Time   `json:"access_token_expires_at"`

}

func (server *Server) renewAccessToken(ctx *gin.Context) {
	var req renewAccessTokenRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	refreshPayload , err := server.tokenMaker.VerifyToken(req.RefreshToken)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}
	
	invalid := errors.New("Session does'nt exist ")


	session , err := server.store.GetRefreshTokenByHash(ctx, req.RefreshToken)

	if err != nil {
	if errors.Is(err, pgx.ErrNoRows) {
		ctx.JSON(http.StatusUnauthorized, errorResponse(invalid))
		return
	}

	handleDBError(ctx, err)
	return
	}



	if session.IsBlocked || session.RevokedAt.Valid {
    ctx.JSON(http.StatusUnauthorized, gin.H{
        "error": "refresh token is no longer valid",
    })
    return
	}

	if session.UserID != refreshPayload.UserID {
    ctx.JSON(http.StatusUnauthorized, gin.H{
        "error": "invalid session",
    })
    return
	}

	if time.Now().After(session.ExpiresAt.Time) {
    ctx.JSON(http.StatusUnauthorized, gin.H{
        "error": "refresh token has expired",
    })
    return
	}

	accessToken , accessPayload , err := server.tokenMaker.CreateToken(
		refreshPayload.Email,
		refreshPayload.UserID,
		refreshPayload.Role,
		server.config.AccessTokenDuration,
	)

	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	ctx.JSON(http.StatusOK, renewAccessTokenResponse{
		AccessToken: accessToken,
		AccessTokenExpiresAt: accessPayload.ExpiredAt,

	})
}