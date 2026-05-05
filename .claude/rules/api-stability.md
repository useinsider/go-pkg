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

## Before opening a PR

- Run `go vet ./...` and `go test ./...` inside the package directory.
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
