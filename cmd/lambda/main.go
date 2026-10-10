// Command lambda is the production entrypoint: it wires the same handler as
// cmd/api (via internal/platform/wiring) but serves it through API Gateway
// HTTP API (payload format 2.0) instead of binding a socket. Config, the
// database pool and the handler are built once at cold start and reused
// across warm invocations in the same execution environment.
//
// Migrations do not run here — a cold start racing another cold start could
// apply them concurrently, and goose gives no guarantee that's safe. They
// run once per deploy via cmd/migrate instead.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"

	"github.com/diegoczajka/travelu-api/internal/config"
	"github.com/diegoczajka/travelu-api/internal/platform/database"
	"github.com/diegoczajka/travelu-api/internal/platform/wiring"
)

// version is overwritten at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	adapter, err := newAdapter()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	lambda.Start(adapter.ProxyWithContext)
}

func newAdapter() (*httpadapter.HandlerAdapterV2, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}

	logger := wiring.NewLogger(cfg)
	slog.SetDefault(logger)

	// Lambda's own init phase has no caller context to cancel on; a bounded
	// context here only protects against a hung connection attempt.
	ctx := context.Background()

	db, err := database.Open(ctx, database.Options{
		DSN:             cfg.DatabaseDSN,
		MaxOpenConns:    cfg.DatabaseMaxOpen,
		MaxIdleConns:    cfg.DatabaseMaxIdle,
		ConnMaxLifetime: cfg.DatabaseConnMaxAge,
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	handler := wiring.BuildHandler(cfg, logger, db, version)
	logger.Info("lambda cold start complete", "version", version, "env", cfg.AppEnv)

	return httpadapter.NewV2(handler), nil
}
