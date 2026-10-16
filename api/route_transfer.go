package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedTransferRoutes(auth gin.IRoutes) {
	auth.POST("/transfers", server.CreateTransfer)
	auth.GET("/transfers", server.ListTransfers)
	auth.GET("/transfers/:id", server.GetTransfer)
	auth.GET("/accounts/:id/transfers", server.ListAccountTransfers)
}
