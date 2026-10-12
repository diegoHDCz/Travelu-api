// Command migrate applies pending goose migrations and exits. It is
// packaged and deployed as its own Lambda function, invoked once per deploy
// by the CI/CD pipeline (via `aws lambda invoke`) before traffic is pointed
// at the new cmd/lambda version — never from the request-serving Lambda's
// cold start, and never from a CI runner directly, so the database
// credentials stay in the Lambda's environment and never reach CI.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"github.com/diegoczajka/travelu-api/internal/config"
	"github.com/diegoczajka/travelu-api/internal/platform/database"
	"github.com/diegoczajka/travelu-api/internal/platform/migrations"
	"github.com/diegoczajka/travelu-api/internal/platform/wiring"
)

const migrationsTimeout = 60 * time.Second

// Response is deliberately tiny: the caller (CI) only needs to know the
// invocation succeeded, since `aws lambda invoke` already surfaces a
// non-2xx/FunctionError as a failed step.
type Response struct {
	Status string `json:"status"`
}

func main() {
	lambda.Start(handleRequest)
}

func handleRequest(ctx context.Context) (Response, error) {
	cfg, err := config.Load()
	if err != nil {
		return Response{}, fmt.Errorf("load config: %w", err)
	}

	logger := wiring.NewLogger(cfg)
	slog.SetDefault(logger)

	db, err := database.Open(ctx, database.Options{
		DSN:             cfg.DatabaseDSN,
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxLifetime: cfg.DatabaseConnMaxAge,
	})
	if err != nil {
		return Response{}, fmt.Errorf("open database: %w", err)
	}
	defer db.Close()

	migrateCtx, cancel := context.WithTimeout(ctx, migrationsTimeout)
	defer cancel()

	if err := migrations.Run(migrateCtx, db.DB); err != nil {
		return Response{}, fmt.Errorf("run migrations: %w", err)
	}

	logger.Info("migrations applied")
	return Response{Status: "ok"}, nil
}
