package api

import "github.com/gin-gonic/gin"

func (server *Server) registerAuthenticatedEntriesRoutes(auth gin.IRoutes) {
	auth.GET("/accounts/:id/entries", server.ListAccountEntries)
	auth.GET("/accounts/:id/entries/:entryId", server.GetAccountEntry)
	// TODO GET /accounts/:id/statement?from=&to=&format=pdf|csv
}

func (server *Server) registerAdminEntriesRoutes(admin gin.IRoutes) {
	admin.GET("/accounts/:id/entries", server.AdminListAccountEntries)
	admin.GET("/entries", server.AdminGetEntriesByReferenceID)
	admin.GET("/accounts/:id/balance-check", server.AdminCheckAccountBalance)
}
