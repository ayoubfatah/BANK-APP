package api

import (
	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
)

func actorFromCtx(ctx *gin.Context) store.Actor {
	payload := getAuthPayload(ctx)

	return store.Actor{
		ID:        payload.UserID,
		Role:      payload.Role,
		IPAddress: ctx.ClientIP(),
		UserAgent: ctx.Request.UserAgent(),
	}
}
