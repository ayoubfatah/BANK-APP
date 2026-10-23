package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedCardsRoute(router gin.IRouter) {
	router.GET("/cards", server.getCards)
	router.GET("/cards/:id", server.getCard)
	router.POST("/cards", server.createCard)
	router.POST("/cards/:id/freeze", server.freezeCard)
	router.POST("/cards/:id/unfreeze", server.UnfreezeCard)
	router.POST("/cards/:id/cancel", server.cancelCard)
	router.POST("/cards/:id/limits", server.changeSpendLimit)
	router.POST("/cards/:id/replace", server.changeSpendLimit)
}
