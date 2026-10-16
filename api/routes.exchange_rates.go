package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedExchangeRateRoutes(auth gin.IRoutes) {
	auth.GET("/exchange-rates", server.GetExchangeRate)
	auth.GET("/exchange-rates/all", server.ListLatestExchangeRates)
	auth.GET("/exchange-rates/history", server.GetExchangeRateHistory)
}

func (server *Server) registerAdminExchangeRateRoutes(admin gin.IRoutes) {
	admin.GET("/exchange-rates", server.AdminListExchangeRates)
}
