# Module boundaries

Every `ins*` directory is an independent Go module with its own `go.mod`.
That's unusual enough to deserve its own rule file — the normal Go
intuitions about imports and refactors don't quite apply.

## No root module

- There is no `go.mod` at the repo root. `go test ./...` from the root
  does nothing useful; always `cd <pkg>` first.
- Editor/LSP integrations need `gopls` with multi-module workspace
  support. A `go.work` file is intentionally not committed — services
  pin specific module versions, and `go.work` would shadow that locally.

## Cross-package imports

- Cross-imports go through the published path:
  `github.com/useinsider/go-pkg/insdash` — not a relative path, not a
  local replace directive.
- Current dependency chain:
  - `inssqs` → `insdash`, `inslogger`
  - `insssm` → `inscacheable`
- Adding a new cross-dep is a design decision, not a casual one. It
  couples release cycles: a breaking change in the dependency forces a
  release of the dependent. Propose the change before coding it.

## No circular deps

- Circular imports across `ins*` modules won't compile and can't be
  recovered without renaming. If two packages want the same type, extract
  a third (smaller) package.

## Shared dep versions

- `scripts/check-deps.sh` lists the expected version of every third-party
  dep across all modules. When bumping, update `check-deps.sh` and every
  `ins*/go.mod` in the same PR. It is a manual check: no workflow runs it,
  it only scans top-level `ins*/` (not `insrequester/v3`), and it needs
  bash 4+. Today `insrequester` v2/v3 are on testify `v1.11.1` against the
  script's `v1.8.1` pin.
- New third-party deps should be weighed carefully. Every dep we pin
  becomes a release coordination burden across 14 modules.

## CI discovers modules

- `.github/workflows/lint.yml` (`golangci-lint` check) and
  `scripts/coverage.sh` (`unit-tests` check) both loop over
  `find . -name go.mod`. A new module needs no workflow change and must not
  get its own job or matrix entry — the two check names are the
  branch-protection contexts.
- The lint config is one file, `.golangci.yaml` at the repo root, passed
  with `--config` from inside each module.

## Release order

- Release a dependency *before* the dependent. Tag `insdash/v1.2.0`, push
  it, then bump `inssqs/go.mod` to require `v1.2.0`, test, and tag
  `inssqs/v1.1.0`. Doing it in the other order breaks `go get` for
  anyone who happens to pull between the two tags.
