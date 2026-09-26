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
