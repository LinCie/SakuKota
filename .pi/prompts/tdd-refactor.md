---
description: Refactor backend production code without changing behavior
argument-hint: "<green handoff>"
---
Perform only the REFACTOR phase using this GREEN handoff: $ARGUMENTS

Read relevant production code and tests. Improve readability and maintainability, remove duplication, simplify, or optimize where justified without changing observable behavior. Test files are read-only: do not modify, delete, weaken, skip, or bypass them. Do not add functionality, trade correctness for elegance, or remove behavior because it appears unnecessary.

Run the complete relevant test suite. Treat failures as regressions and correct production code while preserving tests. Report the refactoring, files changed, results, implementation decisions, and remaining risks. Stop and wait for VERIFICATION.
