# CLAUDE.md — go-pkg

## Project Overview
Shared Go utility library used across Insider's backend services. Multi-module monorepo — each package (`ins*`) is an independent Go module with its own `go.mod` and version tags. The library is consumed by many services, so every exported API is a long-lived contract and backward compatibility is the default posture.

## Repository Structure
```
go-pkg/
├── inscacheable/   # TTL cache wrapper
├── inscodeerr/     # HTTP error codes
├── insdash/        # Utility functions
├── insgorm/        # GORM database wrapper
├── inskinesis/     # AWS Kinesis client (aws-sdk-go v1)
├── inslogger/      # Zap logger wrapper
├── insredis/       # Redis client
├── insrequester/   # HTTP client with retry/circuit breaker — module path .../insrequester/v2 (goresilience)
│   └── v3/         # Separate module .../insrequester/v3 (failsafe-go); same API shape, fix both
├── inssentry/      # Sentry integration
├── inssimpleroute/ # Simple HTTP router
├── inssql/         # SQL client (database/sql)
├── inssqs/         # AWS SQS client (aws-sdk-go-v2)
├── insssm/         # AWS SSM parameter store
├── prompts/        # AI-bot context: test-coverage.md, dependencies.md
├── scripts/        # coverage.sh (CI unit-tests job), check-deps.sh (manual dep pin check)
├── .github/workflows/  # lint.yml, unit-tests.yml, AI review/coverage/security, MySQL 5.x block
└── .golangci.yaml  # One golangci-lint config shared by all 14 modules
```

14 modules in total: the 13 top-level `ins*` directories plus `insrequester/v3`.

## Development Commands
```bash
# Work within a specific package directory — there is no root go.mod
cd <package>
go mod tidy
go test -race ./...
go vet ./...
golangci-lint run --config ../.golangci.yaml ./...   # ../../.golangci.yaml inside insrequester/v3

# Cross-module dep version check (manual; not run by CI)
./scripts/check-deps.sh
```

golangci-lint is v2.12.2 locally and in CI
(`go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2`).
The config is the v2 schema; do not add a v1-format file.

## CI checks

| Check | Workflow | Trigger | Required |
|---|---|---|---|
| `golangci-lint` | `.github/workflows/lint.yml` | push (develop, master) + PR | **yes** (repo ruleset `dataforce-develop-merge-guard`), runs on `ubuntu-latest` |
| `unit-tests` | `.github/workflows/unit-tests.yml` | push | BLOCKED — see below |
| `AI Code Review` | `.github/workflows/ai-code-review.yml` | PR | no |
| `AI Test Coverage` | `.github/workflows/ai-test-coverage.yml` | PR | no |
| `AI Security Review` | `.github/workflows/ai-security-review.yml` | PR + issue comment | no |
| `Security AllInOne` | `.github/workflows/security_allinone.yml` | `feature/*` push + PR | no |
| `Block MySQL 5.x Usage` | `.github/workflows/mysql-version-check.yml` | PR to `develop` | no |

**go-pkg is a public repository, and public repositories get no self-hosted
runner** (DevOps policy, confirmed 2026-09-23). Every self-hosted workflow here
(`Unit Tests`, `Security AllInOne`, `Block MySQL 5.x Usage`) queues until cancelled, and the AI review
workflows fail at startup because they call reusable workflows in a private
repository. `golangci-lint` is the one required check, so `lint.yml` runs on the
GitHub-hosted `ubuntu-latest` runner; it needs no secrets and no internal network.
Do not move `unit-tests.yml` there as-is: its Coverus upload targets an internal
host. The long-term fix is making the repository internal/private.

There is no root module. Both gates run per module:

    root=$(git rev-parse --show-toplevel)
    for m in $(find . -name go.mod | xargs -n1 dirname); do
      (cd "$m" && golangci-lint run --config "$root/.golangci.yaml" ./... && go test -race ./...)
    done

Both `lint.yml` and `scripts/coverage.sh` discover modules with
`find . -name go.mod`, so a new `ins*` module is picked up without a workflow
edit. `golangci-lint` is a single non-matrix job on purpose: its check-run name
is the branch-protection context, and a matrix would rename it per module.

## Key Conventions
- **Multi-module repo**: No root `go.mod`. Each package is independent.
- **Versioning**: Tags follow `<module>/v<version>` format (e.g., `insredis/v1.0.1`). Major bumps change the module path to `/<pkg>/vN+1/`.
- **Dependencies between modules**: `inssqs` depends on `insdash`, `inslogger`. `insssm` depends on `inscacheable`. Release dependencies first.
- **Commit messages**: Prefix with module name (e.g., `inslogger: description of change`).
- **Branching**: Default branch is `develop`. PRs target `develop`.
- **MySQL 5.x blocked**: CI rejects any MySQL 5.x references in PRs.
- **Third-party deps pinned across modules**: see `scripts/check-deps.sh`; update it in the same PR as any version bump. It is a manual check — no workflow runs it.
- **Linting**: `.golangci.yaml` at the repo root is the shared DataForce Go linter set and applies to every module. Every finding is fixed in code except the ones listed under `linters.exclusions.rules`, each with a comment. revive's `var-naming` is exempt for **exported** identifiers only (`Id`, `Url`, `MockSql`, ...) because renaming them is a breaking change; unexported names are still linted. Never add a `disabled:` list under `settings.revive.rules` — in golangci-lint v2 that replaces revive's whole rule set and silently turns the linter off.

## Releasing
```bash
git tag <module>/v1.0.1
git push origin <module>/v1.0.1
gh release create <module>/v1.0.1 --title "<module> v1.0.1" --notes "Description"
```

See [RELEASING.md](RELEASING.md) and [CONTRIBUTING.md](CONTRIBUTING.md) for the full flow.

## Claude Code Automations

**Agents** (subagents for review tasks):
- `.claude/agents/code-reviewer.md` — Reviews for API consistency, module independence, backward compatibility.

**Skills** (invoke with `/skill-name`):
- `/add-package <name>` — Scaffold a new `ins*` package with go.mod, interface, tests, README.
- `/release-package <package> <version>` — Release workflow with dependency chain awareness.

**Hooks** (automatic):
- `PostToolUse` — `.claude/hooks/gofmt-on-write.sh` formats edited `.go` files with `gofmt -s` and `goimports`.
- `PreToolUse` — `.claude/hooks/block-env-files.sh` refuses edits to `.env*`, `*.pem`, credential files, and service-account JSON.

## Testing
Tests use `testify` for assertions and `go-sqlmock`/mocks for database testing. Run tests per-package with `-race`, not from the repo root. CI's `unit-tests` job runs `scripts/coverage.sh`, which executes `go test ./... -count=1 -coverprofile` in every module, merges the profiles (generated `*_mock.go` files excluded) and uploads them to Coverus.

## Rule Imports

The following rule files auto-load into context. Keep them short and
concern-scoped; add a new file rather than growing an existing one.

### Code style
@.claude/rules/code-style.md

### API stability & semver
@.claude/rules/api-stability.md

### Backward compatibility
@.claude/rules/backward-compatibility.md

### Testing
@.claude/rules/testing.md

### Error handling
@.claude/rules/error-handling.md

### Module boundaries
@.claude/rules/module-boundaries.md
