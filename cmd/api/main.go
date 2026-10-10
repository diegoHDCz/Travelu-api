// Command api is the travelu-api standalone HTTP server, for local
// development: it wires config, database, migrations and the shared
// handler from internal/platform/wiring by hand, with graceful shutdown.
// Production runs internal/platform/wiring behind cmd/lambda instead.
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

	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/config"
	"github.com/diegoczajka/travelu-api/internal/platform/database"
	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/diegoczajka/travelu-api/internal/platform/wiring"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	shutdownTimeout   = 15 * time.Second
	migrationsTimeout = 30 * time.Second
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	logger := wiring.NewLogger(cfg)
	slog.SetDefault(logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Open(ctx, database.Options{
		DSN:             cfg.DatabaseDSN,
		MaxOpenConns:    cfg.DatabaseMaxOpen,
		MaxIdleConns:    cfg.DatabaseMaxIdle,
		ConnMaxLifetime: cfg.DatabaseConnMaxAge,
	})
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	if err := runMigrations(ctx, db); err != nil {
		return err
	}

	handler := wiring.BuildHandler(cfg, logger, db, version)

	server := &http.Server{
		Addr:              listenAddr(cfg),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	return serve(ctx, server, logger, cfg)
}

func runMigrations(ctx context.Context, db *sqlx.DB) error {
	migrateCtx, cancel := context.WithTimeout(ctx, migrationsTimeout)
	defer cancel()

	if err := migrations.Run(migrateCtx, db.DB); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// listenAddr binds to localhost only in production, since Caddy is the only
// intended entry point and runs on the same host. Only cmd/api binds like
// this; cmd/lambda never opens a socket.
func listenAddr(cfg config.Config) string {
	host := "0.0.0.0"
	if cfg.IsProduction() {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s:%d", host, cfg.Port)
}

func serve(ctx context.Context, server *http.Server, logger *slog.Logger, cfg config.Config) error {
	serverErr := make(chan error, 1)
	go func() {
		logger.Info("starting server", "addr", server.Addr, "version", version, "env", cfg.AppEnv)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
			return
		}
		close(serverErr)
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("server error: %w", err)
	case <-ctx.Done():
		logger.Info("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("graceful shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
