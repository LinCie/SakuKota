# Go Coding Guidelines

These guidelines follow the Google Go Style Guide (guide, decisions, best practices), Effective Go, and the Go Code Review Comments wiki. Where they conflict, the Go Style Guide takes precedence. Existing code that deviates is not a reason to deviate in new code.

## Project context

- Read `docs/CONTEXT.md` before planning or changing any code. It covers the product's purpose, the tested backend contract, where design docs and tests disagree, and open questions. Tests are the source of truth when they disagree with it.
- Update `docs/CONTEXT.md` for every change, in the same task, independently and without asking the user for permission.
- Update it only after verification passes: build, `go vet`, and the relevant tests. Record only verified behavior, traced to a test or the user's approved spec, never to production code alone.
- Bump "Last updated" in the header, add a one-line entry to the change log table, and remove resolved items from §10.

## Principles

In order of importance: clarity, simplicity, concision, maintainability, consistency.

- Write code for the reader. Clarity beats cleverness.
- Use the simplest mechanism that works: a core language construct first, then the standard library, then an existing dependency, and only then a new abstraction or dependency.
- Do not add abstractions, configuration, or dependencies for hypothetical future needs.
- Keep the normal code path unindented. Handle errors and edge cases first, with early returns.
- Within a package or file, stay consistent with the surrounding code.

## Tooling

Before finishing a change, the code must pass:

- `gofmt -s` (or `goimports`), which is mandatory. Never hand-format.
- `go vet ./...`
- `go build ./...`
- `go test ./...` and, for concurrent code, `go test -race ./...`
- A linter such as `staticcheck` or `golangci-lint`, if the project configures one.

## Formatting

- Use `gofmt` output. Do not fight it.
- Imports go in groups, separated by blank lines: standard library, then everything else. Side-effect imports (`_ "pkg"`) go last and only in `main` packages or tests.
- Never use dot imports (`import . "pkg"`), except the single case where a `_test` package must pretend to be part of a package it cannot otherwise import.
- Rename imports only to resolve a collision, or to avoid a local variable shadowing a package (e.g. `urlpkg`). Prefer the most local or project-specific import to rename.
- There is no fixed line length. Do not split a line just because it is long. Break lines only where the semantics call for it.
- Keep function signatures on one line.
- Do not break `if`, `for`, or `switch` conditions across lines.
- Use the signal-boosting comment for uncommon-looking checks, such as `if err == nil { // if NO error`.

## Naming

- Use MixedCaps and mixedCaps. Never use underscores or `ALL_CAPS` in identifiers. Test function names (`TestX_Y`) are the exception.
- Constants are MixedCaps and named for their role, not their value. Do not prefix them with `K`.
- Initialisms keep one consistent case: `URL`, `ID`, `HTTP`, `API`, `XMLAPI`, `userID`, `ServeHTTP`. Never `Url`, `Id`, or `ServeHttp`.
- Package names are short, lowercase, one word, with no underscores. Avoid `util`, `common`, `helper`, `misc`, `model`, and `types`. Name a package by what it provides, and think about the call site: `sql.Open`, `strings.Cut`.
- Do not stutter. Names read as `pkg.Name`, so write `user.Store`, not `user.UserStore`. A constructor for a package's main type is `New` (`widget.New`), not `widget.NewWidget`.
- Functions and methods do not repeat the package name, receiver type, parameter names, or return types. Use `Parse`, not `ParseYAMLConfig`, in `package yamlconfig`.
- Do not use a `Get` prefix. Name accessors after the noun (`Counts()`, not `GetCounts()`). Use `Fetch`, `Compute`, or `Load` for work that may block or fail.
- Use the verb form for functions that act and the noun form for functions that return something.
- When two functions differ only by type, put the type at the end (`ParseInt`, `ParseInt64`). Mark the primary version by omitting the suffix.
- Variable name length matches scope. Short names are fine in small scopes (`i`, `r`, `w`, `ctx`, `err`). Use longer, descriptive names for package-level and widely used identifiers.
- Do not encode types in names: `users`, not `userSlice`; `count`, not `numUsers`. Add a qualifier only when two versions of the same value coexist (`ageRaw` and `age`).
- Receivers are one or two letters, an abbreviation of the type (`c` for `Client`), used consistently across all methods of that type. Never `this`, `self`, or `me`. Omit an unused receiver name.
- Do not name variables after standard packages (`url`, `http`, `strings`) in any scope that needs the package.
- Sentinel errors are named `ErrXxx`. Error types end in `Error`.
- Interfaces with one method are often named by the method plus `-er` (`Reader`, `Stringer`).
- Use `%q` for printing strings in messages and tests.

## Packages and API design

- Group code by cohesive concept. Split a package when it is doing two things and clients need only one of them. Do not split a package when clients need both types together.
- Use one file per cohesive concept, not one file per type. Put the package comment in exactly one file, typically `doc.go` for long comments.
- Export only what clients need. Exported identifiers are a maintenance contract.
- Make the zero value useful where practical. Use `var x T` for values ready to use without initialization.
- Prefer concrete types. Accept interfaces where a function needs only behavior, and return concrete types.
- Define interfaces in the package that consumes them, not in the package that implements them. Keep them small, with the methods the consumer actually uses.
- Do not create an interface before it has a real need: two or more implementations, or a genuine dependency-break. Do not export test doubles or mocks from the production package; generated mocks live in a `mock` subpackage (see Testing). Test through the real public API.
- Use an option struct when a function takes many arguments, or when most callers set only a few fields. Use functional options for large, evolving APIs. Do not pass a `context.Context` inside an option struct.
- Avoid global mutable state, including package-level registries, singletons, and `init`-time side effects. Pass dependencies explicitly through constructors, function parameters, or struct fields.
- Parse flags only in `package main`. Libraries take configuration through Go APIs, not by registering flags.
- Use generics only when they remove real duplication across multiple concrete types. Do not use them for a single type, and do not use them to build DSLs or heavy abstractions.
- Use type definitions (`type T1 T2`). Use type aliases (`type T1 = T2`) only for package migrations.

## Functions and methods

- Use `context.Context` as the first parameter, named `ctx`. Take it from the caller. Use `context.Background()` only in `main`, `init`, or tests. In tests, prefer `t.Context()`.
- Do not store a `context.Context` in a struct. Pass it to each method that needs it.
- Do not create custom context types or accept context-like interfaces.
- Pass pointers only when mutation or the value's size calls for it. Do not pass `*string` or `*io.Reader` just to save bytes.
- Choose a method receiver type consistently per type. Default to pointer receivers. Use value receivers for small immutable values (`time.Time`), and for slices, maps, funcs, and chans that are not reassigned. Use pointer receivers for types containing a `sync.Mutex` or other no-copy field.
- Use channel direction in parameters (`<-chan T`, `chan<- T`) where the function only sends or receives.
- Prefer synchronous functions. A caller can add concurrency; removing concurrency from a caller is hard.
- Name result parameters only when the names clarify meaning (for example, two results of the same type) or when a deferred function must modify them. Do not name results just to enable naked returns. Use naked returns only in very short functions.
- Do not use `break` at the end of a `switch` case. Use a labeled `break` to exit a loop from inside a `switch`.
- Do not use `else` after a `return`, `continue`, `break`, or `panic`. Handle the error first and let the normal path continue unindented.

## Errors

- Return `error` as the last result. Return `nil` for success and treat other results as unspecified when the error is non-nil.
- Never discard an error with `_` without a comment explaining why it is safe.
- Handle each error exactly once: handle it, return it, or (only in `main`) exit. Do not log and return the same error.
- Keep the error path first and indented, and the success path at the outer level.
- Error strings are lowercase, with no trailing punctuation or newline: `fmt.Errorf("open config: %w", err)`, not `"Open config."`. Capitalize only proper nouns, acronyms, and exported identifiers.
- Each wrap adds context that the callee does not already provide. Describe the action that failed: `load user 42: %w`. Do not repeat information the underlying error already includes, such as a file path from `os`.
- Do not write stop words like "failed to", "unable to", or "could not" in wrapped errors.
- Wrap with `%w` at the end of the message (`"load user: %w"`) so the text reads in order. Use `%w` when callers should inspect the cause with `errors.Is` or `errors.As`. Use `%v` for new errors and to hide internals at API or system boundaries.
- Do not annotate an error solely to say it failed. Return it unchanged.
- Callers check errors with `errors.Is` and `errors.As`. Never compare error strings, and never match on `err.Error()`.
- Export sentinel errors (`var ErrNotFound = errors.New("not found")`) and structured error types only when callers need to distinguish them. Document them in the function's doc comment.
- Return `error`, not a concrete error type, from exported functions. A nil concrete pointer converted to an interface is not nil.
- Use `errors.New` for static messages and `fmt.Errorf` for formatted ones.
- For in-band failures, return a second value (`value, ok bool`, or `error`) instead of sentinel values like `-1` or `""`.
- Use `panic` only for programmer errors and impossible states. Do not use `panic` for normal error flow. Do not recover from panics except at package boundaries for internal parsers that must stay private.
- Use `log.Fatal` or `os.Exit` only in `main`. Initialization code that cannot continue should return an error to `main`.
- `MustXxx` functions are acceptable only for package-level variables initialized at startup (for example, `regexp.MustCompile`). Do not use them on user input or request paths.

## Concurrency

- Make every goroutine's lifetime obvious. Every goroutine must have a clear way to stop: a context, a closed channel, or a `sync.WaitGroup` / `errgroup.Group`. If a goroutine's end is not obvious, document when and why it exits.
- Do not start goroutines inside library code that outlive the call without an owner or cancellation.
- Only the sender should close a channel. Use directional channel types.
- Prefer channels to share data between goroutines, and mutexes to protect state. Keep the locking scope short and inside the function that owns the state.
- Use `errgroup` (`golang.org/x/sync/errgroup`) to coordinate related goroutines that can fail together.
- Never copy a value containing `sync.Mutex`, `sync.WaitGroup`, or `bytes.Buffer` after first use.
- Use `crypto/rand` for keys, tokens, and any security-sensitive randomness. Never use `math/rand` for them.

## Logging and output

- Use `log/slog` for structured logging in services. Use it with key-value pairs, not formatted messages.
- Use error-level logs sparingly, and only for actionable problems. Log at the edge where the error is finally handled.
- Do not log personal or secret data.
- Use `fmt.Fprintf` to write directly to an `io.Writer`, not `fmt.Sprintf` followed by a write.
- Use `strings.Builder` to build strings piece by piece. Use `+` for a few strings. Use `fmt.Sprintf` for formatting.
- Use raw string literals for multi-line constant text.

## Language idioms

- Use composite literals for values with known contents. For types from another package, always use field names in struct literals.
- Use `:=` for variables initialized to a non-zero value, and `var` for zero values. Use `new` or `&T{}` for pointers to zero values, and composite literals when you know the initial values.
- Prefer nil slices (`var s []T`) over empty non-nil ones (`[]T{}`). Check emptiness with `len(s) == 0`, not `s == nil`. Do not make APIs that depend on the difference.
- Preallocate with `make` only when the size is known and performance matters. Do not add size hints by default.
- Use `any`, not `interface{}`.
- Use `for range` loops and single-letter index names where scope is small. Prefer `for i := range n` (Go 1.22+).
- Compare a variable to a constant with the variable on the left (`if result == "foo"`).
- Use `t.Run` subtests, `t.Cleanup`, `t.TempDir`, and `t.Setenv` to keep tests self-contained.
- Use `defer` to release resources immediately after acquiring them (`defer f.Close()`). Check errors that matter from `Close` on writes.
- Do not use `init()` for logic that can fail or depends on configuration.

## Documentation and comments

- Every exported name has a doc comment that begins with its name and is a full sentence ending with a period: `// Encode writes the JSON encoding of req to w.` Use `// Package x ...` for package comments, placed directly above the `package` clause with no blank line.
- Document unexported names when their behavior is not obvious. Begin the comment with the name.
- Comments explain why, not what. Do not restate code. Document non-obvious behavior: concurrency safety, cleanup requirements, context cancellation behavior, and error conditions.
- Document whether a type is safe for concurrent use when it is not obvious.
- Document cleanup requirements explicitly (for example, "Call Stop to release resources").
- Add runnable `Example` functions in `_test.go` files for exported APIs that need usage context.
- Wrap comment lines to a readable width, consistent within a file.

## Testing

- Use the standard `testing` package with `github.com/stretchr/testify` (`assert` to keep checking after a mismatch, `require` when continuing would be meaningless, such as setup failure or a nil result that would panic) and `go.uber.org/mock/gomock` for mocks. Do not add other assertion libraries or test frameworks.
- Mocks must be generated with `mockgen` in source mode, never written by hand. Put a `//go:generate mockgen -source=service.go -destination=mock/service.go -package=mock` line above the interface, in the file that defines it, so the mock lands in the `mock` subpackage of the package that consumes the interface. Commit the generated files, never edit them, and regenerate with `go generate ./...` when an interface changes. Tests of the consumer live in `package foo_test`, since `mock` imports `foo`.
- Create the controller with `gomock.NewController(t)` (it verifies expectations on cleanup). Match arguments precisely; use `gomock.Any()` only for values the test does not care about, such as `ctx`.
- Put tests in `foo_test.go`. Use `package foo` for white-box tests and `package foo_test` for black-box tests of the exported API.
- Name test functions `TestXxx`, benchmarks `BenchmarkXxx`, and fuzz tests `FuzzXxx`.
- Use table-driven tests for many cases with the same logic. Name each case with `t.Run` using short names without spaces or slashes.
- Compare with `assert.Equal`, `assert.ErrorIs`, `assert.JSONEq` and similar, passing `want` (expected) before `got`. Add a message with the input only where the failure would otherwise be ambiguous, such as inside a table loop.
- Mark test helpers with `t.Helper()`. Helpers that only set up or clean up resources should use `require`. Do not write helpers that assert correctness; return values or errors instead, and assert in the test function.
- Do not call `require` or `t.FailNow` from goroutines other than the test goroutine. Use `assert` there.
- Do not compare error message strings. Use `ErrorIs`, `ErrorAs`, or a boolean check for whether an error occurred.
- Keep setup scoped to the tests that need it. Use `TestMain` only for expensive setup shared by all tests in a package, and tear it down explicitly.
- Keep tests hermetic and independent. Do not depend on test order, and do not share mutable package state between tests.
- Put fixtures in `testdata/`.
- Test behavior through public APIs rather than through exported test doubles.
- Add a test for each bug fix that would have caught the bug.

## Scope

- Make the smallest change that satisfies the request. Do not refactor unrelated code while making a change.
- For a new package, add package docs and at least one example or test that shows intended use.
- Preserve existing public API behavior unless the task changes it. If a change alters public behavior, update the relevant doc comments and tests in the same change.
