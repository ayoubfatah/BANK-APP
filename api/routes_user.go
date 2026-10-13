package api

import "github.com/gin-gonic/gin"

func (server *Server) registerPublicUserRoutes(router gin.IRoutes) {
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.POST("/users/logout", server.logoutUser)
}

func (server *Server) registerAuthenticatedUserRoutes(router gin.IRoutes) {
	router.GET("/users/me", server.getUser)
	router.PATCH("/users/me", server.updateUser)
	router.PATCH("/users/phone", server.updateUserPhone)
	router.PATCH("/users/email", server.updateUserEmail)
	router.PATCH("/users/password", server.updateUserPassword)
	router.POST("/users/close", server.closeUserAccount)
}

func (server *Server) registerAdminUserRoutes(admin gin.IRoutes) {
	admin.GET("/users", server.listUsers)
	admin.GET("/users/:id", server.getUserByAdmin)
	admin.PATCH("/users/:id/status", server.setUserStatus)
	admin.PATCH("/users/:id", server.closeUserAccountByAdmin)
}
