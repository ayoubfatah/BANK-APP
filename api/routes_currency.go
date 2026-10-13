package api

import "github.com/gin-gonic/gin"

func (server *Server) registerPublicCurrencyRoutes(router gin.IRoutes) {
	router.GET("/currencies", server.listActiveCurrencies)
	router.GET("/currencies/:code", server.getCurrencyByCode)
}

func (server *Server) registerAdminCurrencyRoutes(admin gin.IRoutes) {
	admin.GET("/currencies", server.listAllCurrencies)
	admin.PATCH("/currencies/:code", server.setCurrencyActive)
}
