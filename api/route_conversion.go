package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedConversionRoutes(auth gin.IRoutes) {
	auth.POST("/conversions/quote", server.CreateConversionQuote)
	auth.POST("/conversions", server.CreateConversion)
	auth.GET("/conversions", server.ListConversions)
	auth.GET("/conversions/:id", server.GetConversion)
}

func (server *Server) registerAdminConversionRoutes(admin gin.IRoutes) {
	admin.GET("/conversions", server.AdminListConversions)
	admin.GET("/conversions/:id", server.AdminGetConversion)
	admin.POST("/conversions/:id/reverse", server.AdminReverseConversion)
}
