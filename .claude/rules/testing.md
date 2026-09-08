# Testing — go-pkg

## Layout

- Tests live alongside code: `<pkg>/<name>_test.go`. Use `package ins<name>`
  (white-box) for access to unexported helpers; `package ins<name>_test`
  (black-box) when testing the public surface — prefer black-box when
  possible since it's what callers see.
- Each package has its own `go.mod`, so tests run per-package:
  `cd <pkg> && go test -race ./...`. Never `go test ./...` from the repo
  root — there is no root module.
- CI: the `unit-tests` check (`.github/workflows/unit-tests.yml`, on push)
  runs `scripts/coverage.sh`, which does `go test ./... -count=1
  -coverprofile` in every module found by `find . -name go.mod`, merges the
  profiles (committed `*_mock.go` files excluded) and uploads to Coverus. A
  failing test in any module fails the check. The `golangci-lint` check
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
  (`unparam`). `bodyclose` is excluded for `_test.go` files because the
  requester tests exercise error paths where the response is nil by
  contract.

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
