# Testing — go-pkg

## Layout

- Tests live alongside code: `<pkg>/<name>_test.go`. Use `package ins<name>`
  (white-box) for access to unexported helpers; `package ins<name>_test`
  (black-box) when testing the public surface — prefer black-box when
  possible since it's what callers see.
- Each package has its own `go.mod`, so tests run per-package:
  `cd <pkg> && go test -race ./...`. Never `go test ./...` from the repo
  root — there is no root module.
- Cross-package end-to-end tests live in the `test/integration` module
  (`cd test/integration && go test ./...`), which wires several `ins*`
  packages together against a REAL dependency — a `redis:7.4-alpine`
  container the workflow starts, dialled at `REDIS_ADDR` (default
  `localhost:6378`). Never stub that boundary and never `t.Skip` when the
  container is absent: `Integration Tests` will be a required check, so a
  skip is a silent pass. (It is BLOCKED today — go-pkg has no runner-group
  access, PA-40353. CLAUDE.md's check table is the single source of truth on
  that status.) Single-package tests stay next to their code and stay
  docker-free.
- CI: the `Unit Tests` check (`.github/workflows/unit-tests.yml`, on push)
  runs `scripts/coverage.sh`, which does `go test ./... -count=1
  -coverprofile` in every module found by `find . -name go.mod` *except*
  `test/integration`, merges the profiles (committed `*_mock.go` files
  excluded) and uploads to Coverus. The `Integration Tests` check
  (`.github/workflows/integration-tests.yml`, on push) runs the
  `test/integration` module on its own. A failing test in any module fails
  its check. The `golangci-lint` check
  lints test files with the same `.golangci.yaml` as production code.

## Frameworks

- **`testify`** (`github.com/stretchr/testify`) for assertions.
  `scripts/check-deps.sh` pins `v1.8.1`; `insrequester` (v2) and
  `insrequester/v3` are already on `v1.11.1`, and the script is neither run
  by CI nor able to see `insrequester/v3` (it loops top-level `ins*/` only;
  it also needs bash 4+ for `declare -A`, so on macOS's bash 3.2 it prints
  syntax errors and a false "all match"). Treat it as advisory.
- **`golang/mock`** (`insredis`, `insrequester` v2) or **`go.uber.org/mock`**
  (`inskinesis`, `inssqs`, `insrequester/v3`) for interface mocks. Mock
  files are generated and committed — CI does not regenerate.
- **`go-sqlmock`** (`github.com/DATA-DOG/go-sqlmock`) for `inssql` /
  `insgorm` database tests. No real MySQL container in unit tests.

## Structure

- Table-driven tests with `t.Run(tt.name, ...)` subtests. Name cases after
  the condition being tested, not the expected output.
- Assert the minimum: one behaviour per subtest. Mixing "it returns X
  and also logs Y" into one assertion block makes failure messages
  useless.
- When testing error paths, assert on the error *value* (`errors.Is` against
  a sentinel) or the wrapping structure, not the string. A value from
  `recover()` is `interface{}`: assert it to `error` first.
- Lint rules that bite in tests: helpers taking `*testing.T` start with
  `t.Helper()` (`thelper`); type assertions use the two-value form
  (`forcetypeassert`); unused handler params are `_` (`revive`); helper
  parameters that every caller passes the same value for get removed
  (`unparam`). `bodyclose` is excluded only for the two `insrequester`
  modules' test files (about 60 call sites, many on error paths where the
  response is nil by contract, so each would need its own conditional
  close — not worth the churn in a lint PR); every other module's tests
  must close response bodies.
- Tests against a local `httptest.Server` must not use a client timeout in
  the low-millisecond range unless the timeout *is* what is being tested:
  a 1 ms deadline can beat the response under `-race` and turns a
  deterministic assertion into a ~5% flake (seen in both `insrequester`
  modules' "last error" cases and fixed in PA-40353). Likewise, never
  assert a fixed order on a slice built from a map — use
  `assert.ElementsMatch`.

## Fakes vs. mocks

- Use a generated mock when you need to verify call order or arguments.
- Use a hand-rolled fake (a small in-memory implementation of `Interface`)
  when you just need the behaviour. Fakes survive signature changes in the
  interface; mocks don't.

## Coverage expectation

- New exported functions must have a test. Coverage isn't measured at a
  threshold, but unexercised public API is a red flag in review.
- Concurrency-sensitive code (retry loops, circuit breakers, cache
  eviction) needs `go test -race` green.
