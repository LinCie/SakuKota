.PHONY: dev dev-setup dev-backend dev-frontend start build postgres-up postgres-down redis-up storage-up storage-init cleanup migrate-up migrate-down migrate-status migrate-version migrate-reset migration seed seed-bulk test test-integration

-include .env

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/rantaugo?sslmode=disable
MIGRATIONS_DIR ?= migrations
# Pass everything from .env (JWT_SECRET, PORT, ...) on to the commands below.
.EXPORT_ALL_VARIABLES:
# Set CONTAINER_ENGINE=podman (in .env or the environment) to use Podman.
CONTAINER_ENGINE ?= docker
COMPOSE ?= $(CONTAINER_ENGINE) compose
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.28.0

dev-backend:
	@command -v air >/dev/null || { echo "air not found"; exit 1; }
	air

dev-frontend:
	@command -v pnpm >/dev/null || { echo "pnpm not found"; exit 1; }
	@test -d web/node_modules || pnpm --dir web install
	pnpm --dir web dev

# Run backend (air) and frontend (vite) together; stopping either (or Ctrl-C) stops both.
dev:
	@command -v air >/dev/null || { echo "air not found"; exit 1; }
	@command -v pnpm >/dev/null || { echo "pnpm not found"; exit 1; }
	@test -d web/node_modules || pnpm --dir web install
	@bash -c 'trap "trap - EXIT; kill \$$(jobs -p) 2>/dev/null; wait" INT TERM EXIT; \
		air & pnpm --dir web dev & \
		wait -n'

# One-time local setup: start the services, then migrate, init storage, and seed
# (the seed uploads images).
dev-setup:
	$(MAKE) postgres-up
	@until $(COMPOSE) exec -T postgres pg_isready -U $${POSTGRES_USER:-postgres} >/dev/null 2>&1; do echo "waiting for postgres..."; sleep 1; done
	$(MAKE) migrate-up
	$(MAKE) storage-up
	@until $(COMPOSE) exec -T garage /garage status >/dev/null 2>&1; do echo "waiting for garage..."; sleep 1; done
	$(MAKE) storage-init
	$(MAKE) seed
	$(MAKE) redis-up

start:
	go run ./cmd/api

build:
	go build -o ./bin/api ./cmd/api

postgres-up:
	$(COMPOSE) up -d postgres

postgres-down:
	$(COMPOSE) down

redis-up:
	$(COMPOSE) up -d redis

storage-up:
	$(COMPOSE) up -d garage

storage-init:
	bash docker/garage/init.sh

# Delete one batch of storage objects queued for deletion.
cleanup:
	go run ./cmd/cleanup

migrate-up:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate up

migrate-down:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate down

migrate-status:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate status

migrate-version:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate version

migrate-reset:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/migrate reset

seed:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed

# 50,000 generated places over 17 kecamatan, for load testing only.
seed-bulk:
	DATABASE_URL="$(DATABASE_URL)" go run ./cmd/seed-bulk

migration:
	@test -n "$(name)" || (echo "usage: make migration name=create_users" && exit 1)
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" postgres "$(DATABASE_URL)" create "$(name)" sql

# Unit tests. They need no database or container engine.
test:
	go test ./...

# Unit tests plus the repository tests, which start a PostgreSQL container with
# testcontainers, so Docker must be running.
test-integration:
	go test -tags integration ./...
