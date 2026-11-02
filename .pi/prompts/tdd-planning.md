---
description: Plan the backend behavioral contract
argument-hint: "<requirements>"
---
Perform only the PLANNING phase for: $ARGUMENTS

Read `docs/CONTEXT.md` and relevant tests. You may read production code only to describe exported names and signatures, never as evidence of intended behavior. Do not write repository files.

Decompose requirements into normal, boundary, invalid-input, failure, and state-transition scenarios; identify invariants, side effects, and concurrency guarantees. Ask the user about ambiguities rather than inventing requirements.

Deliver a behavioral specification and requirement-to-scenario matrix. Include inputs, outputs, errors, state and side-effect guarantees, and the relevant API contract (package paths, exported names, signatures) in the RED handoff. Include everything RED needs from context; RED cannot read it. Stop and wait for the next phase.
