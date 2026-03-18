# CLAUDE.md — go-pkg

## Project Overview
Shared Go utility library used across Insider's backend services. Multi-module monorepo — each package (`ins*`) is an independent Go module with its own `go.mod` and version tags.

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
└── scripts/
```

## Development Commands
```bash
# Work within a specific package directory
cd <package>
go mod tidy
go test ./...
```

## Key Conventions
- **Multi-module repo**: No root `go.mod`. Each package is independent.
- **Versioning**: Tags follow `<module>/v<version>` format (e.g., `insredis/v1.0.1`).
- **Dependencies between modules**: `inssqs` depends on `insdash`, `inslogger`. `insssm` depends on `inscacheable`. Release dependencies first.
- **Commit messages**: Prefix with module name (e.g., `inslogger: description of change`).
- **Branching**: Default branch is `develop`. PRs target `develop`.
- **MySQL 5.x blocked**: CI rejects any MySQL 5.x references in PRs.

## Releasing
```bash
git tag <module>/v1.0.1
git push origin <module>/v1.0.1
gh release create <module>/v1.0.1 --title "<module> v1.0.1" --notes "Description"
```

## Claude Code Automations

**Agents** (subagents for review tasks):
- `.claude/agents/code-reviewer.md` — Reviews for API consistency, module independence, backward compatibility

**Skills** (invoke with `/skill-name`):
- `/add-package <name>` — Scaffold a new `ins*` package with go.mod, interface, tests, README
- `/release-package <package> <version>` — Release workflow with dependency chain awareness

**Hooks** (automatic):
- Go files auto-formatted with `gofmt` on edit

## Testing
Tests use `testify` for assertions and `go-sqlmock`/mocks for database testing. Run tests per-package, not from the repo root.
