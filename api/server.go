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
	server.registerPublicAccountDetailsRoutes(auth)
	server.registerAuthenticatedEntriesRoutes(auth)

	// Admin routes
	admin := router.Group("/admin")
	admin.Use(authMiddleware(server.tokenMaker))
	admin.Use(adminMiddleware())

	server.registerAdminUserRoutes(admin)
	server.registerAdminKYCRoutes(admin)
	server.registerAdminCurrencyRoutes(admin)
	server.registerAdminAccountRoutes(admin)
	server.registerAdminAccountDetailsRoutes(admin)
	server.registerAdminEntriesRoutes(admin)

	server.router = router
}

func (server *Server) Start(address string) error {
	return server.router.Run(address)
}
