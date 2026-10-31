# Project Overview

Rantau ("rantaugo") is a Go REST API with a React/Vite frontend. The Go module is `rantau-go`; the backend uses PostgreSQL with PostGIS.

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

## Backend Development (TDD)

Backend changes follow test-driven development (Red → Green → Refactor). This applies to Go code under `cmd/` and `internal/`, not the `web/` frontend.

1. **Red** — Before implementing, write tests that validate the business logic and confirm they fail for the expected reason.
2. **Green** — Write the simplest code that makes the tests pass. Do not add behavior the tests do not require.
3. **Refactor** — With the tests green, improve the code's structure and readability without changing behavior. Re-run the tests after refactoring.

- Test business logic in the domain's business logic layer; keep tests independent of HTTP and the database where possible.
- Follow the existing Go testing style in the relevant area.
- Each phase counts as one explicit step. Do not move on to the next phase until the user asks for it.

## Task Execution

- Do only what is explicitly asked in the current turn. Do not expand scope, refactor adjacent code, or make extra changes.
- Work incrementally. Each turn is one explicit step; do not perform later or implied steps. Wait for the next explicit instruction.
- Apply requested changes even if they break existing code (e.g., changed signatures with mismatched callers, failing tests, red diagnostics, build errors). Do not fix the resulting breakage.
- Always report every failure, error, or build break caused or revealed by the work, with file paths and error messages, even when not fixed.
- The agent is a helper and implementor of intent, not a builder. Do not make design decisions beyond the stated request.
