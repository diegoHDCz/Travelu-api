BINARY := dist/api
LAMBDA_DIST := dist/lambda
MIGRATE_DIST := dist/migrate
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMPOSE := docker compose -f docker-compose.dev.yml
DEV_DSN := postgres://app:app@127.0.0.1:5432/travelu?sslmode=disable
MIGRATIONS_DIR := internal/platform/migrations/sql
TF_DIR := deploy/terraform
TF_PLAN := tfplan

.PHONY: run build build-lambda build-migrate test lint migrate-up migrate-down migrate-new db-up db-down tf-init tf-plan tf-apply deploy

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

lint:
	golangci-lint run ./...

tf-init:
	terraform -chdir=$(TF_DIR) init

# The plan reads the zips via filebase64sha256(), so the artifacts must be
# built first. The saved plan file is what tf-apply executes, so what you
# reviewed is exactly what gets applied.
tf-plan: build-lambda build-migrate tf-init
	terraform -chdir=$(TF_DIR) plan -out=$(TF_PLAN)

tf-apply:
	@test -f $(TF_DIR)/$(TF_PLAN) || (echo "no saved plan — run 'make tf-plan' first" >&2; exit 1)
	terraform -chdir=$(TF_DIR) apply $(TF_PLAN)
	rm -f $(TF_DIR)/$(TF_PLAN)

deploy: tf-plan
	@read -p "apply this plan? [y/N] " ans && [ "$$ans" = "y" ]
	$(MAKE) tf-apply

migrate-up:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) postgres "$(DEV_DSN)" up

migrate-down:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) postgres "$(DEV_DSN)" down

migrate-new:
	go run github.com/pressly/goose/v3/cmd/goose -dir $(MIGRATIONS_DIR) create $(name) sql

db-up:
	$(COMPOSE) up -d
	@echo "aguardando o postgres ficar saudável..."
	@until $(COMPOSE) ps postgres | grep -q healthy; do sleep 1; done

db-down:
	$(COMPOSE) down
