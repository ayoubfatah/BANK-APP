package api

import "github.com/gin-gonic/gin"

func (server *Server) registerPublicTokenRoutes(router gin.IRoutes) {
	router.POST("/tokens/renew_access", server.renewAccessToken)
}
