# Testing — go-pkg

## Layout

- Tests live alongside code: `<pkg>/<name>_test.go`. Use `package ins<name>`
  (white-box) for access to unexported helpers; `package ins<name>_test`
  (black-box) when testing the public surface — prefer black-box when
  possible since it's what callers see.
- Each package has its own `go.mod`, so tests run per-package:
  `cd <pkg> && go test ./...`. Never `go test ./...` from the repo root —
  there is no root module.

## Frameworks

- **`testify`** (`github.com/stretchr/testify`) for assertions. Pin to the
  version in `scripts/check-deps.sh` (currently `v1.8.1`).
- **`golang/mock`** or **`go.uber.org/mock`** for interface mocks. Mock
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
  a sentinel) or the wrapping structure, not the string.

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
