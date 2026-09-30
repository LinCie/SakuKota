.PHONY: dev dev-backend dev-frontend start build postgres-up postgres-down migrate-up migrate-down migrate-status migrate-version migrate-reset migration seed

-include .env

DATABASE_URL ?= postgres://postgres:postgres@localhost:5432/rantaugo?sslmode=disable
MIGRATIONS_DIR ?= migrations
export DATABASE_URL MIGRATIONS_DIR
GOOSE := go run github.com/pressly/goose/v3/cmd/goose@v3.27.3

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
	go run ./cmd/web

build:
	go build -o ./bin/web ./cmd/web

postgres-up:
	docker compose up -d postgres

postgres-down:
	docker compose down

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
