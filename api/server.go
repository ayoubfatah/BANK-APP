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
	// kyc
	router.POST("/users/me/kyc", server.closeUserAccount)
	

	server.router = router
	return server
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}

