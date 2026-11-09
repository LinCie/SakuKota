.PHONY: dev dev-backend dev-frontend start build postgres-up postgres-down redis-up storage-up storage-init cleanup migrate-up migrate-down migrate-status migrate-version migrate-reset migration seed

-include .env

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/rantaugo?sslmode=disable
MIGRATIONS_DIR ?= migrations
# Pass everything from .env (JWT_SECRET, PORT, ...) on to the commands below.
.EXPORT_ALL_VARIABLES:
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

start:
	go run ./cmd/api

build:
	go build -o ./bin/api ./cmd/api

postgres-up:
	docker compose up -d postgres

postgres-down:
	docker compose down

redis-up:
	docker compose up -d redis

storage-up:
	docker compose up -d garage

storage-init:
	bash .docker/garage/init.sh

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

migration:
	@test -n "$(name)" || (echo "usage: make migration name=create_users" && exit 1)
	$(GOOSE) -dir "$(MIGRATIONS_DIR)" postgres "$(DATABASE_URL)" create "$(name)" sql
