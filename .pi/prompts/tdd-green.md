---
description: Implement the approved backend behavioral tests
argument-hint: "<red handoff>"
---
Perform only the GREEN phase using this RED handoff: $ARGUMENTS

Read the approved specification, behavioral tests, relevant production code, and `docs/CONTEXT.md`. Implement the smallest reasonable production change that satisfies the contract and tests, following existing architecture where compatible. Do not add intentional behavior without a requirement and test.

Test files are read-only: never modify, delete, weaken, skip, or bypass them. If a test contradicts the requirements or cannot be satisfied, stop and report it for PLANNING/RED; do not reshape the contract around the implementation.

Run relevant tests and report actual results, production changes, implementation decisions, and known limitations. Stop and wait for REFACTOR.
