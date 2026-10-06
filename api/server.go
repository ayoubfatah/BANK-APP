package api

import (
	"github.com/bank-app/db/store"
	"github.com/gin-gonic/gin"
)

type Server struct {
	store  *store.Store
	router *gin.Engine
}

func NewServer(store *store.Store) *Server {
	server := &Server{store: store}
	router := gin.Default()

	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.GET("/users/me", server.getUser)
	router.PATCH("/users/me", server.updateUser)
	router.PATCH("/users/phone", server.updateUserPhone)
	router.PATCH("/users/email", server.updateUserEmail)
	router.PATCH("/users/password", server.updateUserPassword)
	router.POST("/users/close", server.closeUserAccount)

	// admin: users
	router.GET("/admin/users", server.listUsers)                  // new
	router.GET("/admin/users/:id", server.getUserByAdmin)            // new
	router.PATCH("/admin/users/:id/status", server.setUserStatus) // new
	router.PATCH("/admin/users/:id", server.closeUserAccountByAdmin)          // new



	// K Y C 
	router.POST("/users/me/kyc", server.SubmitKYC)
	router.GET("/users/me/kyc", server.getKYCStatus)   
	// admin : 
	router.GET("/admin/kyc", server.ListKYC)
	router.PATCH("/admin/kyc/:id/verify", server.verifyKYC)
	router.PATCH("/admin/kyc/:id/reject", server.rejectKYC)

	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

