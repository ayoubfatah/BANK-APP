package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedBeneficiaryRoutes(auth gin.IRoutes) {
	auth.POST("/beneficiaries", server.CreateBeneficiary)
	auth.GET("/beneficiaries", server.ListBeneficiaries)
	auth.GET("/beneficiaries/:id", server.GetBeneficiary)
	auth.PATCH("/beneficiaries/:id", server.UpdateBeneficiary)
	auth.DELETE("/beneficiaries/:id", server.DeleteBeneficiary)
}
