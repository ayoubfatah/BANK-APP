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

	// Public routes
	server.registerPublicUserRoutes(router)
	server.registerPublicTokenRoutes(router)
	server.registerPublicCurrencyRoutes(router)

	// Authenticated routes
	auth := router.Group("/")
	auth.Use(authMiddleware(server.tokenMaker))

	server.registerAuthenticatedUserRoutes(auth)
	server.registerAuthenticatedKYCRoutes(auth)
	server.registerAuthenticatedAccountRoutes(auth)

	// Admin routes
	admin := router.Group("/admin")
	admin.Use(authMiddleware(server.tokenMaker))
	admin.Use(adminMiddleware())

	server.registerAdminUserRoutes(admin)
	server.registerAdminKYCRoutes(admin)
	server.registerAdminCurrencyRoutes(admin)

	server.router = router
}

func (server *Server) registerPublicTokenRoutes(router gin.IRoutes) {
	router.POST("/tokens/renew_access", server.renewAccessToken)
}
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

func (server *Server) registerAuthenticatedKYCRoutes(router gin.IRoutes) {
	router.POST("/users/me/kyc", server.submitKYC)
	router.GET("/users/me/kyc", server.getKYCStatus)
}

func (server *Server) registerAdminUserRoutes(admin gin.IRoutes) {
	admin.GET("/users", server.listUsers)
	admin.GET("/users/:id", server.getUserByAdmin)
	admin.PATCH("/users/:id/status", server.setUserStatus)
	admin.PATCH("/users/:id", server.closeUserAccountByAdmin)
}

func (server *Server) registerAdminKYCRoutes(admin gin.IRoutes) {
	admin.GET("/kyc", server.listKYC)
	admin.PATCH("/kyc/:id/verify", server.verifyKYC)
	admin.PATCH("/kyc/:id/reject", server.rejectKYC)
}

func (server *Server) registerPublicCurrencyRoutes(router gin.IRoutes) {
	router.GET("/currencies", server.listActiveCurrencies)
	router.GET("/currencies/:code", server.getCurrencyByCode)
}

func (server *Server) registerAdminCurrencyRoutes(admin gin.IRoutes) {
	admin.GET("/currencies", server.listAllCurrencies)
	admin.PATCH("/currencies/:code", server.setCurrencyActive)
}
func (server *Server) registerAuthenticatedAccountRoutes(admin gin.IRoutes) {
	admin.POST("/accounts", server.CreateAccount)
	admin.POST("/accounts/:id/freeze", server.FreezeAccount)
	admin.GET("/accounts", server.GetAccounts)
	admin.GET("/accounts/:id", server.GetAccount)

}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}