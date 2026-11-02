---
description: Write backend behavioral tests from an approved specification
argument-hint: "<planning handoff>"
---
Perform only the RED phase using this approved PLANNING handoff: $ARGUMENTS

The RED agent MUST NOT READ ANY FILE OTHER THAN `*_test.go` FILES.

RED requires a fresh agent context that has not read any non-test file during the task. If this context has already read a non-test file during the task, state that RED isolation is not satisfied and ask the user how to proceed. When delegating, include the full PLANNING handoff and the restriction above verbatim.

- Locate test files only using commands that list `*_test.go` paths; read, create, and modify only those files. You may run relevant `go test` commands.
- Do not read production files, `go.mod`, migrations, testdata, docs, or config, or inspect implementation indirectly using tools, history, or subagents. Do not derive expected behavior from implementation or compiler errors. Report missing or differing APIs as mismatches rather than adapting tests.
- Translate the approved contract into precise, deterministic behavioral tests covering normal cases, boundaries, errors, invariants, and side effects, including negative assertions for prohibited effects. Prefer external-package black-box tests and business-logic tests independent of HTTP and database where possible.
- Change existing tests only for an approved requirement change, a test that misrepresents the approved requirement, or a resolved ambiguity; justify each change. Never change tests to make implementation pass.
- Fix test defects, syntax errors, and wrong assumptions; they are not legitimate RED failures. Report requirements that cannot be expressed due to API ambiguity.

Deliver changed tests, requirement-to-test mapping, actual test results, expected failures, and unresolved API ambiguities. Stop and wait for GREEN.
