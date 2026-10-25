package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bank-app/api"
	"github.com/bank-app/db/store"
	"github.com/bank-app/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	startupTimeout  = 5 * time.Second
	shutdownTimeout = 15 * time.Second
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := utils.LoadConfig(".")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Cancelled on Ctrl+C or SIGTERM (deploys, container stops).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// One timeout covers pool creation and the connectivity check.
	startCtx, cancelStart := context.WithTimeout(ctx, startupTimeout)
	defer cancelStart()
	//  this just setusp the pool doesnt connect it
	pool, err := pgxpool.New(startCtx, config.DBURL)
	if err != nil {
		return fmt.Errorf("create database pool: %w", err)
	}
	// Runs after the server has drained, since run() only returns then.
	defer pool.Close()

	if err := pool.Ping(startCtx); err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	st := store.NewStore(pool)

	server, err := api.NewServer(config, st)
	if err != nil {
		return fmt.Errorf("create server: %w", err)
	}

	serverErr := make(chan error, 1)
	go func() {
		err := server.Start()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("server stopped unexpectedly: %w", err)
		}
		return nil
	case <-ctx.Done():
		slog.Info("shutdown signal received, draining requests")
	}

	// Restore default signal handling so a second Ctrl+C force-quits.
	stop()

	//  here its built from context.Background since we already canceled the ctx above in the stop()
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	slog.Info("shutdown complete")
	return nil
}
