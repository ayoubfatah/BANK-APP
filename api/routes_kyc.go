package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedKYCRoutes(router gin.IRoutes) {
	router.POST("/users/me/kyc", server.submitKYC)
	router.GET("/users/me/kyc", server.getKYCStatus)
}

func (server *Server) registerAdminKYCRoutes(admin gin.IRoutes) {
	admin.GET("/kyc", server.listKYC)
	admin.PATCH("/kyc/:id/verify", server.verifyKYC)
	admin.PATCH("/kyc/:id/reject", server.rejectKYC)
}
