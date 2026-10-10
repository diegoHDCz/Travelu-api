BINARY := dist/api
LAMBDA_DIST := dist/lambda
MIGRATE_DIST := dist/migrate
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMPOSE := docker compose -f docker-compose.dev.yml
DEV_DSN := app:app@tcp(127.0.0.1:3306)/travelu?parseTime=true&loc=UTC&charset=utf8mb4
MIGRATIONS_DIR := internal/platform/migrations/sql

.PHONY: run build build-lambda build-migrate test test-integration lint migrate-up migrate-down migrate-new db-up db-down

run:
	go run ./cmd/api

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) ./cmd/api

# Lambda's custom runtime (provided.al2023) requires the binary to be named
# "bootstrap" and shipped as a zip — this is the artifact Terraform's
# aws_lambda_function.filename points at.
build-lambda:
	mkdir -p $(LAMBDA_DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(LAMBDA_DIST)/bootstrap ./cmd/lambda
	cd $(LAMBDA_DIST) && zip -q -j lambda.zip bootstrap

build-migrate:
	mkdir -p $(MIGRATE_DIST)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags "-X main.version=$(VERSION)" -o $(MIGRATE_DIST)/bootstrap ./cmd/migrate
	cd $(MIGRATE_DIST) && zip -q -j migrate.zip bootstrap

test:
	go test -race ./...

test-integration: db-up
	TEST_DATABASE_DSN="$(DEV_DSN)" go test -race ./...

lint:
	golangci-lint run ./...




migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) mysql "$(DEV_DSN)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) mysql "$(DEV_DSN)" down

migrate-new:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) create $(name) sql

db-up:
	$(COMPOSE) up -d
	@echo "aguardando o mysql ficar saudável..."
	@until $(COMPOSE) ps mysql | grep -q healthy; do sleep 1; done

db-down:
	$(COMPOSE) down
