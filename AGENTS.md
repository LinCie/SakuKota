# Project Overview

Rantau ("rantaugo") is a Go REST API with a React/Vite frontend. The Go module is `rantau-go`; the backend uses PostgreSQL with PostGIS.

## Project context

Read `docs/CONTEXT.md` before planning or changing behavior. It covers the product's purpose, the current tested backend contract, where the design docs and the tests disagree, and open questions. It is a living document: update it only after a VERIFICATION phase passes. Tests remain the source of truth when they disagree with it. CONTEXT.md is not a `*_test.go` file, so the RED agent must not read it; PLANNING includes whatever RED needs in the handoff.

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

# Backend Development: Specification-First, Test-First

This workflow applies to Go code under `cmd/` and `internal/`, not the `web/` frontend.

## Core principles (non-negotiable)

1. Tests are the executable specification of the system's intentional, externally observable behavior.
2. Every intentional production behavior must be justified by a requirement and represented by appropriate tests.
3. If a behavior has not been specified and tested, do not implement it.
4. Production code conforms to the behavioral contract; it never redefines it.
5. Tests verify observable behavior, not implementation details.
6. Production code is never authoritative evidence of intended behavior.
7. No agent may silently introduce, remove, weaken, or change a functional requirement.

The goal is complete functional parity between the behavioral specification and the production code. Parity covers externally observable behavior — outputs, errors, state changes, side effects, persistence, and explicitly specified guarantees. It does not require testing every internal function.

**The test suite defines what the software is allowed and required to do. Production code determines how that behavior is implemented.** Production code, agents, and models are replaceable; the specification and its tests are the durable assets. A task is complete only when the approved contract is implemented and independently verified — not when the code merely compiles or appears to work.

## Phases

Development runs in five phases, in order. Each phase is one explicit step: do not start the next phase until the user asks for it.

| Phase | Agent capability | May read | May write |
|---|---|---|---|
| 1. PLANNING | Strong reasoning | Anything | Nothing in the repo (produces the specification) |
| 2. RED | Strong reasoning | **Only `*_test.go` files** | `*_test.go` files only |
| 3. GREEN | Economical, capable | Production code and tests | Production code only |
| 4. REFACTOR | Economical, capable | Production code and tests | Production code only |
| 5. VERIFICATION | Automated checks; strong reasoning for independent review | Anything | Nothing (reports only) |

### 1. PLANNING

- Understand the user's requirements and decompose them into explicit behavioral scenarios: normal cases, boundaries, invalid inputs, failures, and state transitions.
- Identify invariants, side effects, and concurrency requirements where applicable.
- Establish requirement-to-scenario traceability.
- Identify ambiguities and ask the user; never resolve them silently with assumptions.
- Never infer intended behavior from existing production code. Production code may be read only to describe the current API surface (exported names and signatures) for the handoff.
- Do not write production code or tests.

Deliverable: a behavioral specification and scenario matrix.

### 2. RED

**The RED agent MUST NOT READ ANY FILE OTHER THAN `*_test.go` FILES.**

The PLANNING specification is passed to RED directly as task context. That handoff is not permission to read any other repository file.

Allowed:
- Locate test files only with commands that list `*_test.go` paths (e.g. `find . -name '*_test.go'`).
- Read, create, and modify `*_test.go` files, when justified by the approved specification.
- Run relevant tests (`go test ...`) and report results.

Forbidden:
- Reading production `.go` files or any other non-test file (including `go.mod`, migrations, `testdata/`, docs, and config).
- Inspecting implementation by any other means (`grep`, `cat`, `go doc`, `gopls`, editor tooling, git history, subagents, etc.).
- Modifying any non-test file.
- Deriving expected behavior from existing implementation, including from compiler error output. If a test does not compile because the specified API is missing or differs, report it as an API mismatch; do not adapt the test to whatever the code currently exposes.

RED must:
- Translate the specification into executable tests covering normal behavior, edge cases, error handling, invariants, and side effects.
- Prefer black-box tests through exported APIs (external `package <name>_test`, as existing tests do); test business logic in the domain's business logic layer, independent of HTTP and the database where possible.
- Include negative assertions where needed to prove prohibited side effects do not occur.
- Keep tests precise and deterministic, following the existing Go testing style.
- Report requirements that cannot yet be expressed because the API contract is ambiguous.

RED is complete when the tests accurately represent the approved contract. Tests may fail because functionality is not implemented yet; test defects, syntax errors, and wrong assumptions are not legitimate RED failures and must be fixed in RED.

Deliverable: behavioral tests, test results, and requirement-to-test traceability.

### 3. GREEN

- Read production code and the behavioral tests; implement the smallest reasonable change that satisfies the tests.
- Follow existing architecture and conventions where compatible with the specification.
- Run the relevant tests and report actual results; never hide failures.
- Test files are READ-ONLY: do not modify, delete, weaken, skip, or bypass tests.
- Do not add intentional functionality without a corresponding requirement and test, and do not change expected behavior to suit an implementation.
- If a test appears wrong or contradicts the requirements, stop and report it; it returns to PLANNING/RED.

Deliverable: a working implementation that satisfies the behavioral contract.

### 4. REFACTOR

- Improve readability and maintainability, remove duplication, simplify, and optimize where justified — without changing observable behavior.
- Test files are READ-ONLY. Do not change the contract, add functionality, trade correctness for elegance, or remove behavior because it looks unnecessary.
- Run the complete relevant test suite afterwards. A failing test is a regression: correct the production code and keep the tests unchanged.

Deliverable: improved production code with unchanged observable behavior.

### 5. VERIFICATION

Verify that:
- Every requirement has corresponding tests, and every intentional production behavior traces to a requirement and a test.
- All tests pass, and no test was weakened, skipped, or bypassed.
- No unapproved functional behavior was introduced.
- Error handling and negative behavior are verified, and no regressions occurred.
- Mutation testing is used where practical to judge fault detection.

Typical commands (no Make targets exist for these):
- `go test ./...`
- `go test -race ./...`
- `go vet ./...`
- `go test -tags integration ./...` — integration tests; needs the compose services (`make postgres-up`, `make storage-up`).
- `go test -cover ./...` or `-coverprofile` for coverage analysis.
- Mutation testing only where a tool is available; none is configured in this repo, so do not add one without the user's approval.

Coverage percentages alone never prove behavioral completeness.

Deliverable: verification results and any unresolved issues.

## Agent separation

- When the user asks for phases to be delegated, run each phase in its own agent instance. Spawning agents still requires the user's explicit request.
- RED must run in a fresh agent context that has not read any non-test file during the task. The RED prompt must contain the PLANNING handoff and the RED restriction verbatim. If the current agent has already read non-test files and is asked to perform RED itself, it must state that RED isolation is not satisfied and ask the user how to proceed.
- The implementation agent must not shape the specification because an implementation would be easier.
- Separate model instances are not automatically independent; independence comes from information boundaries, explicit responsibilities, and verification.

## Changing tests

Test changes are controlled specification changes, made only in RED, and only when:
- an approved requirement changes,
- an existing test misrepresents the approved requirement, or
- a genuine ambiguity has been resolved.

Every test change must be justified against the specification. GREEN and REFACTOR never change tests. Never modify a test to make an implementation pass; if the specification needs correcting, return to PLANNING and RED.

## Handoffs

Each phase hands off explicitly; do not rely on undocumented conversational assumptions.

- **PLANNING → RED:** behavioral requirements, scenario matrix, expected inputs and outputs, error contracts, state and side-effect guarantees, relevant API contract (package paths, exported names, signatures).
- **RED → GREEN:** tests created or modified, requirement-to-test mapping, test results, expected failures, unresolved API ambiguities.
- **GREEN → REFACTOR:** production changes, test results, known limitations, implementation decisions.
- **REFACTOR → VERIFICATION:** refactoring summary, files changed, test results, remaining risks.

## Ambiguities and failures

Never invent requirements.

- **Incomplete requirements:** identify the ambiguity, explain its behavioral implications, ask the user or return to PLANNING, update the specification, then update tests in RED before implementing.
- **Implementation cannot satisfy a test:** investigate the implementation, check the contract, never weaken the test, and escalate contradictions to PLANNING.
- **Refactoring breaks tests:** treat it as a regression, fix the production code, and preserve the tests.

# Task Execution

- Do only what is explicitly asked in the current turn. Do not expand scope, refactor adjacent code, or make extra changes.
- Work incrementally. Each turn is one explicit step; do not perform later or implied steps. Wait for the next explicit instruction.
- Apply requested changes even if they break existing code (e.g., changed signatures with mismatched callers, failing tests, red diagnostics, build errors). Do not fix the resulting breakage. Exception: in GREEN and REFACTOR, making the relevant tests pass and fixing regressions caused by the phase's own changes are part of the requested step.
- Always report every failure, error, or build break caused or revealed by the work, with file paths and error messages, even when not fixed.
- The agent is a helper and implementor of intent, not a builder. Do not make design decisions beyond the stated request; behavioral decisions belong to the user via PLANNING, and GREEN/REFACTOR must report any internal implementation decisions they make.
