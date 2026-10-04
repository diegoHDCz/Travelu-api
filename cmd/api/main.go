// Command api is the travelu-api HTTP server: a single monolithic binary
// wiring config, database, migrations, domains and the HTTP server by hand.
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

	"github.com/diegoczajka/travelu-api/internal/auth"
	"github.com/diegoczajka/travelu-api/internal/booking"
	"github.com/diegoczajka/travelu-api/internal/category"
	"github.com/diegoczajka/travelu-api/internal/config"
	"github.com/diegoczajka/travelu-api/internal/listing"
	"github.com/diegoczajka/travelu-api/internal/platform/database"
	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/diegoczajka/travelu-api/internal/review"
	"github.com/diegoczajka/travelu-api/internal/tripdate"
	"github.com/diegoczajka/travelu-api/internal/user"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "dev"

const (
	maxRequestBody    = 1 << 20 // 1 MB
	shutdownTimeout   = 15 * time.Second
	migrationsTimeout = 30 * time.Second
	healthPingTimeout = 2 * time.Second
	loginRatePerSec   = 1.0
	loginRateBurst    = 5
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

	logger := newLogger(cfg)
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

	handler := buildHandler(cfg, logger, db)

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

// buildHandler wires every domain by hand and returns the fully
// middleware-wrapped HTTP handler.
func buildHandler(cfg config.Config, logger *slog.Logger, db *sqlx.DB) http.Handler {
	userRepo := user.NewMySQLRepository(db)
	userService := user.NewService(userRepo)

	refreshTokenRepo := auth.NewMySQLRefreshTokenRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(userService, refreshTokenRepo, tokenManager, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService)

	registerLimiter := auth.NewIPRateLimiter(loginRatePerSec, loginRateBurst, cfg.TrustProxy)
	loginLimiter := auth.NewIPRateLimiter(loginRatePerSec, loginRateBurst, cfg.TrustProxy)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(db))
	mux.Handle("POST /api/v1/auth/register", registerLimiter.Middleware()(http.HandlerFunc(authHandler.Register)))
	mux.Handle("POST /api/v1/auth/login", loginLimiter.Middleware()(http.HandlerFunc(authHandler.Login)))
	mux.HandleFunc("POST /api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", authHandler.Logout)
	mux.Handle("GET /api/v1/users/me", auth.RequireAuth(tokenManager)(http.HandlerFunc(authHandler.Me)))

	requireAuth := auth.RequireAuth(tokenManager)

	categoryRepo := category.NewMySQLRepository(db)
	categoryService := category.NewService(categoryRepo)
	category.NewHandler(categoryService).RegisterRoutes(mux, requireAuth)

	bookingRepo := booking.NewMySQLRepository(db)
	bookingService := booking.NewService(bookingRepo)
	booking.NewHandler(bookingService).RegisterRoutes(mux, requireAuth)

	listingRepo := listing.NewMySQLRepository(db)
	listingService := listing.NewService(listingRepo)
	listing.NewHandler(listingService).RegisterRoutes(mux, requireAuth)

	reviewRepo := review.NewMySQLRepository(db)
	reviewService := review.NewService(reviewRepo)
	review.NewHandler(reviewService).RegisterRoutes(mux, requireAuth)

	tripDateRepo := tripdate.NewMySQLRepository(db)
	tripDateService := tripdate.NewService(tripDateRepo)
	tripdate.NewHandler(tripDateService).RegisterRoutes(mux, requireAuth)

	return httpx.Chain(mux,
		httpx.Recover(),
		httpx.RequestID(),
		httpx.AccessLog(logger),
		httpx.CORS(cfg.CORSAllowedOrigins),
		httpx.MaxBodyBytes(maxRequestBody),
	)
}

// healthHandler is used by the deploy process to decide whether to roll
// back: it pings the database and reports 503 if that fails.
func healthHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), healthPingTimeout)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			httpx.WriteJSON(w, http.StatusServiceUnavailable, map[string]string{
				"status":  "unavailable",
				"version": version,
			})
			return
		}

		httpx.WriteJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"version": version,
		})
	}
}

// listenAddr binds to localhost only in production, since Caddy is the only
// intended entry point and runs on the same host.
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

func newLogger(cfg config.Config) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLogLevel(cfg.LogLevel)}

	var handler slog.Handler
	if cfg.IsProduction() {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	return slog.New(handler)
}

func parseLogLevel(s string) slog.Level {
	var level slog.Level
	if err := level.UnmarshalText([]byte(s)); err != nil {
		return slog.LevelInfo
	}
	return level
}
