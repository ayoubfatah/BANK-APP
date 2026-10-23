package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedAccountDetailsRoutes(router gin.IRoutes) {
	router.GET("/accounts/:id/details", server.GetAccountDetails)
}

func (server *Server) registerAdminAccountDetailsRoutes(admin gin.IRoutes) {
	// POST /admin/accounts/:id/details
	admin.PATCH("/admin/accounts/:id/details", server.UpdateAccountDetails)

}
