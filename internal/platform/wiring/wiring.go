// Package wiring builds the fully middleware-wrapped HTTP handler shared by
// every entrypoint that serves the API (the standalone cmd/api server and
// the cmd/lambda adapter). Keeping this in internal/ instead of cmd/api lets
// both entrypoints import it — a Go "package main" cannot be imported from
// another command.
package wiring

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/diegoczajka/travelu-api/internal/auth"
	"github.com/diegoczajka/travelu-api/internal/booking"
	"github.com/diegoczajka/travelu-api/internal/category"
	"github.com/diegoczajka/travelu-api/internal/config"
	"github.com/diegoczajka/travelu-api/internal/listing"
	"github.com/diegoczajka/travelu-api/internal/platform/apidocs"
	"github.com/diegoczajka/travelu-api/internal/platform/audit"
	"github.com/diegoczajka/travelu-api/internal/platform/httpx"
	"github.com/diegoczajka/travelu-api/internal/review"
	"github.com/diegoczajka/travelu-api/internal/tripdate"
	"github.com/diegoczajka/travelu-api/internal/user"
)

const (
	maxRequestBody    = 1 << 20 // 1 MB
	healthPingTimeout = 2 * time.Second
	loginRatePerSec   = 1.0
	loginRateBurst    = 5
)

// BuildHandler wires every domain by hand and returns the fully
// middleware-wrapped HTTP handler. version is baked in by the caller's
// -ldflags "-X main.version=..." and surfaced on /health.
func BuildHandler(cfg config.Config, logger *slog.Logger, db *sqlx.DB, version string) http.Handler {
	userRepo := user.NewPostgresRepository(db)
	userService := user.NewService(userRepo)

	refreshTokenRepo := auth.NewPostgresRefreshTokenRepository(db)
	tokenManager := auth.NewTokenManager(cfg.JWTSecret, cfg.AccessTokenTTL)
	authService := auth.NewService(userService, refreshTokenRepo, tokenManager, cfg.RefreshTokenTTL)
	authHandler := auth.NewHandler(authService)

	registerLimiter := auth.NewIPRateLimiter(loginRatePerSec, loginRateBurst, cfg.TrustProxy)
	loginLimiter := auth.NewIPRateLimiter(loginRatePerSec, loginRateBurst, cfg.TrustProxy)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", healthHandler(db, version))
	apidocs.RegisterRoutes(mux)
	mux.Handle("POST /api/v1/auth/register", audit.Action("auth.register", registerLimiter.Middleware()(http.HandlerFunc(authHandler.Register))))
	mux.Handle("POST /api/v1/auth/login", audit.Action("auth.login", loginLimiter.Middleware()(http.HandlerFunc(authHandler.Login))))
	mux.Handle("POST /api/v1/auth/refresh", audit.Action("auth.refresh", http.HandlerFunc(authHandler.Refresh)))
	mux.Handle("POST /api/v1/auth/logout", audit.Action("auth.logout", http.HandlerFunc(authHandler.Logout)))
	mux.Handle("GET /api/v1/users/me", audit.Action("user.get_me", auth.RequireAuth(tokenManager)(http.HandlerFunc(authHandler.Me))))

	requireAuth := auth.RequireAuth(tokenManager)

	categoryRepo := category.NewPostgresRepository(db)
	categoryService := category.NewService(categoryRepo)
	category.NewHandler(categoryService).RegisterRoutes(mux, requireAuth)

	bookingRepo := booking.NewPostgresRepository(db)
	bookingService := booking.NewService(bookingRepo)
	booking.NewHandler(bookingService).RegisterRoutes(mux, requireAuth)

	listingRepo := listing.NewPostgresRepository(db)
	listingService := listing.NewService(listingRepo)
	listing.NewHandler(listingService).RegisterRoutes(mux, requireAuth)

	reviewRepo := review.NewPostgresRepository(db)
	reviewService := review.NewService(reviewRepo)
	review.NewHandler(reviewService).RegisterRoutes(mux, requireAuth)

	tripDateRepo := tripdate.NewPostgresRepository(db)
	tripDateService := tripdate.NewService(tripDateRepo)
	tripdate.NewHandler(tripDateService).RegisterRoutes(mux, requireAuth)

	return httpx.Chain(mux,
		httpx.Recover(),
		httpx.RequestID(),
		httpx.AccessLog(logger),
		audit.Middleware(audit.Config{
			Logger:    logger,
			RequestID: httpx.RequestIDFrom,
			ClientIP:  func(r *http.Request) string { return httpx.ClientIP(r, cfg.TrustProxy) },
			Skip:      skipAudit,
		}),
		httpx.CORS(cfg.CORSAllowedOrigins),
		httpx.MaxBodyBytes(maxRequestBody),
	)
}

// skipAudit keeps health checks (polled constantly by CI and monitors) and
// the static API docs out of the audit trail; they carry no user action.
func skipAudit(r *http.Request) bool {
	return r.URL.Path == "/health" || strings.HasPrefix(r.URL.Path, "/docs")
}

// healthHandler reports whether the database is reachable. Behind the
// standalone server it is what the systemd-era deploy process used to decide
// a rollback; behind Lambda it is what CI polls through API Gateway after a
// deploy, and what CloudWatch/Route53 health checks can watch continuously.
func healthHandler(db *sqlx.DB, version string) http.HandlerFunc {
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

// NewLogger builds the shared slog.Logger: JSON in production (so it reads
// well in CloudWatch Logs Insights and the old systemd/journald setup
// alike), plain text in development.
func NewLogger(cfg config.Config) *slog.Logger {
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
