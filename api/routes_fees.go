package api

import "github.com/gin-gonic/gin"

func (server *Server) registerPublicFeeRoutes(router gin.IRoutes) {
	router.GET("/fees/schedule", server.GetFeeSchedule)
}

func (server *Server) registerAuthenticatedFeeRoutes(auth gin.IRoutes) {
	auth.GET("/transfers/:id/fees", server.ListTransferFees)
}

func (server *Server) registerAdminFeeRoutes(admin gin.IRoutes) {
	admin.GET("/fee-rules", server.AdminListFeeRules)
	admin.POST("/fee-rules", server.AdminCreateFeeRule)
	admin.PATCH("/fee-rules/:id", server.AdminUpdateFeeRule)
	admin.GET("/fees", server.AdminListFees)
}
