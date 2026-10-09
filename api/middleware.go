package api

import (
	"net/http"
	"strings"

	"github.com/bank-app/token"
	"github.com/gin-gonic/gin"
)
const (
 authorizationHeaderKey        = "Authorization"
 authorizationHeaderBearerType = "bearer"
 authorizationPayloadKey = "authorization_payload"
)

func authMiddleware(maker token.Maker) gin.HandlerFunc {
 return func(ctx *gin.Context) {
  authHeader := ctx.GetHeader(authorizationHeaderKey)
  if authHeader == "" {
   ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "No header was passed"})
   return
  }

  fields := strings.Fields(authHeader)
  if len(fields) != 2 {
   ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or Missing Bearer Token"})
   return
  }

  authType := fields[0]
  if strings.ToLower(authType) != authorizationHeaderBearerType {
   ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization Type Not Supported"})
   return
  }

  token := fields[1]
  payload, err := maker.VerifyToken(token)
  if err != nil {
   ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Access Token Not Valid"})
   return
  }
  ctx.Set(authorizationPayloadKey , payload)
  ctx.Next()
 }
}


func getAuthPayload(ctx *gin.Context) *token.Payload {
	return ctx.MustGet(authorizationPayloadKey).(*token.Payload)
}


func adminMiddleware() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		if !strings.EqualFold(getAuthPayload(ctx).Role, "admin") {
			ctx.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Admin access required"})
			return
		}
		
		ctx.Next()
	}
}