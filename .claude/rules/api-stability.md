# API stability & semver

This repo is a shared library. Every exported symbol is a contract with
every downstream service. Breaking one breaks many — often silently at
build-time for whoever upgrades next, which is always someone else.

## What counts as a breaking change

Any of these requires a new major version tag (`<pkg>/v<N+1>.0.0`) and
moving the module path to `/<pkg>/vN+1/`:

- Removing or renaming an exported identifier (func, type, const, var,
  method, field).
- Changing a function signature — parameters, return types, receiver type.
- Adding a required method to an exported interface. (Adding optional
  methods via a new interface is fine.)
- Changing observable behaviour of an existing API — default timeouts,
  retry counts, whether a nil input is accepted, etc.
- Changing Go version floor in `go.mod` when callers pin older toolchains.
- Swapping a third-party dep for an incompatible one that leaks through the
  public API (e.g. `*redis.Client` from v6 vs v8).

## What is safe

- Adding new exported types, funcs, constants.
- Adding new fields to a `Config` struct that default sensibly to the
  previous behaviour when zero-valued.
- Adding methods to a concrete (unexported) type.
- Internal refactors that keep signatures + behaviour identical.
- New packages (`ins<new>/`) — they're independent modules.

## Semver rules in this repo

- Tags are `<module>/v<major>.<minor>.<patch>`. The leading `<module>/` is
  not decoration — Go's module proxy routes on it.
- Patch (`vX.Y.Z+1`): bug fixes only, no API change.
- Minor (`vX.Y+1.0`): new additions, strictly backward-compatible.
- Major (`vX+1.0.0`): anything breaking, and the module path gains `/vN`.
  Example: `insrequester/v2`. Callers migrate by updating imports.

## Exported names and the linter

`.golangci.yaml` enables revive, whose `var-naming` rule wants `Id` -> `ID`,
`Url` -> `URL`, `Sql` -> `SQL`. For **exported** identifiers that rename is
exactly the breaking change described above, so the config excludes
`var-naming` for exported struct fields, types, funcs, methods, consts and
vars (see the `linters.exclusions.rules` entry and its comment). Known
survivors, kept on purpose: `inssqs.SQSMessageEntry.Id`,
`MessageDeduplicationId`, `MessageGroupId`, `Config.EndpointUrl`,
`sqs.API.GetQueueUrl`, `inssql.MockSql`, and `inscodeerr.CodeErr` (which
`errname` would call `CodeError`, excluded the same way). Unexported names
are linted normally and were renamed (`getQueueUrl` -> `getQueueURL`, etc.).

The exclusion lives in `linters.exclusions.rules`, never as a `disabled:`
list under `settings.revive.rules` — in golangci-lint v2 that list replaces
revive's entire rule set and switches the linter off silently.

## Before opening a PR

- Run `golangci-lint run --config <repo-root>/.golangci.yaml ./...`,
  `go vet ./...` and `go test -race ./...` inside the package directory.
  CI runs the same lint config as the `golangci-lint` check and the tests
  as the `unit-tests` check.
- If you touched an exported symbol, run `gorelease -base=<last-tag>` (or
  eyeball the diff) and confirm the planned version bump is correct.
- If a dependent `ins*` module needs updating too, note the release order
  in the PR: dependencies first, then dependents. See
  `CONTRIBUTING.md` for the current chain.

## Deprecation over removal

When an API needs to change, prefer a deprecation cycle: add the new
symbol, mark the old one with a `// Deprecated:` comment pointing at the
replacement, and remove only on the next major. Silent removal forces
every caller to fix build errors on our schedule, not theirs.
