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
  `ins*/go.mod` in the same PR — mismatches fail CI.
- New third-party deps should be weighed carefully. Every dep we pin
  becomes a release coordination burden across 13 modules.

## Release order

- Release a dependency *before* the dependent. Tag `insdash/v1.2.0`, push
  it, then bump `inssqs/go.mod` to require `v1.2.0`, test, and tag
  `inssqs/v1.1.0`. Doing it in the other order breaks `go get` for
  anyone who happens to pull between the two tags.
