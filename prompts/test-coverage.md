# go-pkg Test Coverage Guidelines

## Repository

`go-pkg` is Insider's shared, public Go utility library, consumed by many
backend services. It is a **multi-module monorepo**: each `ins*` package
(`insredis`, `inssqs`, `inslogger`, `insgorm`, `inssql`, `insrequester`, etc.)
is an independent Go module with its own `go.mod`. There is no root module.
Every exported symbol is a long-lived contract — backward compatibility is
the default.

## Output Discipline

- Only flag coverage gaps for **non-test `*.go` source** added or changed in
  this PR. Do not request tests for code outside the PR scope.
- Tests run **per package** (`cd <pkg> && go test ./...`), never from the repo
  root — there is no root module. Tie each requirement to its module.

## Test Conventions

- Tests live alongside code: `<pkg>/<name>_test.go`. Use `package ins<name>`
  (white-box) for unexported helpers; prefer `package ins<name>_test`
  (black-box) for the public surface, since that's what callers see.
- **Table-driven** tests with `t.Run(tt.name, ...)` subtests; one behaviour
  per subtest. Name cases after the condition, not the expected output.
- Frameworks: `testify` for assertions; `golang/mock` / `go.uber.org/mock`
  for interface mocks (committed, not regenerated in CI); `go-sqlmock` for
  `inssql` / `insgorm`. No real DB/AWS in unit tests — use mocks/fakes.
- Error paths: assert on the error value (`errors.Is` against a sentinel) or
  wrapping structure, not the string.
- Concurrency-sensitive code (retry, circuit breaker, cache eviction) must be
  `go test -race` green.

## Focus Coverage On

- **Exported package APIs** — the contract callers depend on. New exported
  functions must have a test; unexercised public API is a review red flag.
- Behavioural logic: retry/backoff, circuit-breaker state, cache TTL/eviction,
  query building, serialization, error wrapping.

## What NOT to Test

- `go.mod` / `go.sum`, generated mock files, CI workflows (`.github/`),
  `scripts/`.
- Thin pass-through wrappers over third-party SDK calls with no added logic.
- Trivial getters/constructors that only assign fields.

**Backward-compatibility warning**: flag any change to an exported signature,
struct field, interface, or error sentinel — it can break downstream services.
