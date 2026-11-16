# Project Overview

Rantau ("rantaugo") is a Go REST API with a React/Vite frontend. The Go module is `rantau-go`; the backend uses PostgreSQL with PostGIS.

## Project context

Read `docs/CONTEXT.md` before planning or changing behavior. It covers the product's purpose, the current tested backend contract, where the design docs and the tests disagree, and open questions. Tests remain the source of truth when they disagree with it.

**Mandatory:** whenever a change alters backend behavior or the API contract, update `docs/CONTEXT.md` in the same task, once VERIFICATION passes (build, vet, and the relevant tests), and add a change log entry. Do not wait for the user to ask. This overrides the scope rule below for that file only.

## What the repository contains

- `cmd/` — API, database migration, and seed entry points
- `internal/module/` — backend features organized by domain
- `internal/` — shared config, HTTP, middleware, routing, database, and PostGIS packages
- `migrations/` — database schema migrations, which also define the categories and their custom fields
- `web/` — standalone React and TypeScript frontend

## Conventions

- Backend features are organized by domain, with HTTP, business logic, and data access kept in separate layers.
- Shared backend infrastructure lives outside the domain modules.
- The frontend is maintained as a separate app in `web/`.
- Migrations MUST be created with `make migration name=<name>` (goose generates the timestamped file). Never create or hand-name migration files in `migrations/`.
- Categories and their custom fields are defined in migrations, not created through the API. The API returns each category's `code`, `label`, and `color` (a palette key), and the frontend maps `color` to its styles.
- Follow the existing Go and TypeScript style in the relevant area.

# Task Execution

- Do only what is explicitly asked in the current turn. Do not expand scope, refactor adjacent code, or make extra changes.
- Work incrementally. Each turn is one explicit step; do not perform later or implied steps. Wait for the next explicit instruction.
- Always report every failure, error, or build break caused or revealed by the work, with file paths and error messages, even when not fixed.
