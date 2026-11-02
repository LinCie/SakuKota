---
description: Independently verify the backend contract and implementation
argument-hint: "<refactor handoff>"
---
Perform only the VERIFICATION phase using this handoff: $ARGUMENTS

Read the approved specification, tests, relevant production code, and `docs/CONTEXT.md`. Do not write repository files. Verify requirement-to-test and intentional-behavior-to-requirement/test traceability; passing tests; no weakened, skipped, or bypassed tests; no unapproved behavior; error and negative behavior; and absence of regressions. Use mutation testing where practical, but do not add a tool without user approval. Coverage alone does not prove completeness.

As applicable, run `go test ./...`, `go test -race ./...`, `go vet ./...`, `go test -tags integration ./...`, and `go test -cover ./...` (or a cover profile). Integration tests require the compose services (`make postgres-up`, `make storage-up`); report unavailable prerequisites and all failures rather than concealing them.

Report verification results and unresolved issues. Do not update `docs/CONTEXT.md` during verification; it may be updated only after verification passes and when explicitly requested.
