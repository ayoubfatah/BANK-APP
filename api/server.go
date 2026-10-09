package api

import (
	"fmt"

	"github.com/bank-app/db/store"
	"github.com/bank-app/token"
	"github.com/bank-app/utils"
	"github.com/gin-gonic/gin"
)

type Server struct {
	config     utils.Config
	store      *store.Store
	tokenMaker token.Maker
	router     *gin.Engine
}

func NewServer(config utils.Config, store *store.Store) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
	}
	server.setupRouter()

	return server, nil
}

func (server *Server) setupRouter() {
	router := gin.Default()

	server.registerUserRoutes(router)
	server.registerKYCRoutes(router)
	server.registerCurrencyRoutes(router)

	server.router = router
}

func (server *Server) registerUserRoutes(router *gin.Engine) {
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)

	router.GET("/users/me", server.getUser)
	router.PATCH("/users/me", server.updateUser)
	router.PATCH("/users/phone", server.updateUserPhone)
	router.PATCH("/users/email", server.updateUserEmail)
	router.PATCH("/users/password", server.updateUserPassword)
	router.POST("/users/close", server.closeUserAccount)

	// Admin
	admin := router.Group("/admin/users")
	admin.GET("", server.listUsers)
	admin.GET("/:id", server.getUserByAdmin)
	admin.PATCH("/:id/status", server.setUserStatus)
	admin.PATCH("/:id", server.closeUserAccountByAdmin)
}

func (server *Server) registerKYCRoutes(router *gin.Engine) {
	router.POST("/users/me/kyc", server.submitKYC)
	router.GET("/users/me/kyc", server.getKYCStatus)

	// Admin
	admin := router.Group("/admin/kyc")
	admin.GET("", server.listKYC)
	admin.PATCH("/:id/verify", server.verifyKYC)
	admin.PATCH("/:id/reject", server.rejectKYC)
}

func (server *Server) registerCurrencyRoutes(router *gin.Engine) {
	router.GET("/currencies", server.listActiveCurrencies)
	router.GET("/currencies/:code", server.getCurrencyByCode)

	// Admin
	admin := router.Group("/admin/currencies")
	admin.GET("", server.listAllCurrencies)
	admin.PATCH("/:code", server.setCurrencyActive)
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}