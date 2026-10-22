# Project Overview

Rantau ("rantaugo") is a Go REST API with a React/Vite frontend. The Go module is `rantau-go`; the backend uses PostgreSQL with PostGIS.

## What the repository contains

- `cmd/` — API and database migration entry points
- `internal/module/` — backend features organized by domain
- `internal/` — shared HTTP, routing, database, and PostGIS packages
- `migrations/` — database schema migrations
- `web/` — standalone React and TypeScript frontend

## Conventions

- Backend features are organized by domain, with HTTP, business logic, and data access kept in separate layers.
- Shared backend infrastructure lives outside the domain modules.
- The frontend is maintained as a separate app in `web/`.
- Follow the existing Go and TypeScript style in the relevant area.

## Task Execution

- Do only what is explicitly asked in the current turn. Do not expand scope, refactor adjacent code, or make extra changes.
- Work incrementally. Each turn is one explicit step; do not perform later or implied steps. Wait for the next explicit instruction.
- Apply requested changes even if they break existing code (e.g., changed signatures with mismatched callers, failing tests, red diagnostics, build errors). Do not fix the resulting breakage.
- Always report every failure, error, or build break caused or revealed by the work, with file paths and error messages, even when not fixed.
- The agent is a helper and implementor of intent, not a builder. Do not make design decisions beyond the stated request.
