package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

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
	httpServer *http.Server
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

	server.httpServer = &http.Server{
		Addr:              config.ServerAddress,
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
	}

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
	server.registerAuthenticatedAccountDetailsRoutes(auth)
	server.registerAuthenticatedEntriesRoutes(auth)
	server.registerAuthenticatedTransferRoutes(auth)
	server.registerAuthenticatedCardsRoute(auth)

	// Admin routes
	admin := router.Group("/admin")
	admin.Use(authMiddleware(server.tokenMaker))
	admin.Use(adminMiddleware())

	server.registerAdminCardsRoute(admin)
	server.registerAdminUserRoutes(admin)
	server.registerAdminKYCRoutes(admin)
	server.registerAdminCurrencyRoutes(admin)
	server.registerAdminAccountRoutes(admin)
	server.registerAdminAccountDetailsRoutes(admin)
	server.registerAdminEntriesRoutes(admin)

	server.router = router
}

func (server *Server) Start(address string) error {
	server.httpServer = &http.Server{
		Addr:              address,
		Handler:           server.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return server.httpServer.ListenAndServe()
}

func (server *Server) Shutdown(ctx context.Context) error {
	if server.httpServer == nil {
		return nil
	}
	return server.httpServer.Shutdown(ctx)
}
