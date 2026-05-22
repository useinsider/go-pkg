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
├── inskinesis/     # AWS Kinesis client
├── inslogger/      # Zap logger wrapper
├── insredis/       # Redis client
├── insrequester/   # HTTP client with retry/circuit breaker
├── inssentry/      # Sentry integration
├── inssimpleroute/ # Simple HTTP router
├── inssql/         # SQL client (database/sql)
├── inssqs/         # AWS SQS client
├── insssm/         # AWS SSM parameter store
└── scripts/        # check-deps.sh and other repo-wide tools
```

## Development Commands
```bash
# Work within a specific package directory — there is no root go.mod
cd <package>
go mod tidy
go test ./...
go vet ./...

# Cross-module dep version check
./scripts/check-deps.sh
```

## Key Conventions
- **Multi-module repo**: No root `go.mod`. Each package is independent.
- **Versioning**: Tags follow `<module>/v<version>` format (e.g., `insredis/v1.0.1`). Major bumps change the module path to `/<pkg>/vN+1/`.
- **Dependencies between modules**: `inssqs` depends on `insdash`, `inslogger`. `insssm` depends on `inscacheable`. Release dependencies first.
- **Commit messages**: Prefix with module name (e.g., `inslogger: description of change`).
- **Branching**: Default branch is `develop`. PRs target `develop`.
- **MySQL 5.x blocked**: CI rejects any MySQL 5.x references in PRs.
- **Third-party deps pinned across modules**: see `scripts/check-deps.sh`; update it in the same PR as any version bump.

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
Tests use `testify` for assertions and `go-sqlmock`/mocks for database testing. Run tests per-package, not from the repo root.

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
