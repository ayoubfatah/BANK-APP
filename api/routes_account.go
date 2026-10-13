package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedAccountRoutes(auth gin.IRoutes) {
	auth.POST("/accounts", server.CreateAccount)
	auth.GET("/accounts", server.GetAccounts)
	auth.GET("/accounts/:id", server.GetAccount)
	auth.POST("/accounts/:id/freeze", server.FreezeAccount)
	auth.POST("/accounts/:id/unfreeze", server.UnFreezeAccount)
	auth.POST("/accounts/:id/close", server.CloseAccount)
}

func (server *Server) registerAdminAccountRoutes(admin gin.IRoutes) {
	admin.GET("/accounts", server.AdminListAccounts)
	admin.GET("/accounts/:id", server.AdminGetAccount)
	admin.POST("/accounts/:id/freeze", server.AdminFreezeAccount)
	admin.POST("/accounts/:id/unfreeze", server.AdminUnfreezeAccount)
	admin.POST("/accounts/:id/close", server.AdminCloseAccount)
}
