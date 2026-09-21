# Code Reviewer

Review code changes for API consistency and library quality in the go-pkg multi-module monorepo.

## Focus Areas

### API Consistency
- New packages export an `Interface` type as the primary contract and a constructor that returns it (`inslogger`, `inssqs` follow this). Older packages keep their historical names (`insredis.RedisInterface`, `insrequester.Requester`, `inskinesis.StreamInterface`, `inssql.New` returning `*sql.DB`, `insgorm.NewGorm` returning `*gorm.DB`) — do not ask for them to be renamed, that is a breaking change
- `Config` struct for options where a package has one
- No breaking changes to existing exported APIs without major version bump

### Linting
- `.golangci.yaml` at the repo root is the shared DataForce set (golangci-lint v2.12.2); CI runs it per module as the `golangci-lint` check. Flag code that the linter would reject, do not restate what it already enforces
- revive's `var-naming` is deliberately excluded for **exported** identifiers (`Id`, `Url`, `MockSql`, `GetQueueUrl`) and `errname` for `inscodeerr.CodeErr`: never request those renames. Unexported names must follow Go initialisms
- Suppressions belong in `linters.exclusions.rules` with a comment; flag inline `//nolint` and any `disabled:` list under `settings.revive.rules`
- `insrequester` (v2) and `insrequester/v3` are the same package shape; a fix in one should be ported to the other

### Module Independence
- Each `ins*` package has its own `go.mod` — no root module
- Cross-package imports must use the published module path (e.g., `github.com/useinsider/go-pkg/insdash`)
- Dependency chain respected: `inssqs` → `insdash`, `inslogger`; `insssm` → `inscacheable`
- No circular dependencies between packages

### Error Handling
- Use `inscodeerr.CodeErr` for HTTP-aware errors where applicable
- Wrap errors with context using `fmt.Errorf("operation: %w", err)` or `pkg/errors`
- Compare with `errors.Is`/`errors.As`, never `==` or a type assertion (`errorlint`)
- Never swallow errors silently
- Exported sentinel error message text is consumer-facing; do not change it in unrelated PRs

### Testing
- Tests use `testify` for assertions
- Mock interfaces generated via `mockgen` (`golang/mock` in older modules, `go.uber.org/mock` in newer ones); committed, not regenerated in CI
- `go-sqlmock` for database testing (insgorm, inssql)
- Tests run per-package: `cd <package> && go test -race ./...`; CI's `Unit Tests` check runs `scripts/coverage.sh` over every module
- Test helpers start with `t.Helper()`; type assertions in tests are checked (`thelper`, `forcetypeassert`)

### Backward Compatibility
- Exported functions, types, and interfaces must not be removed without a major version
- New optional fields use functional options or config structs
- Default behavior must not change in minor/patch versions

### Documentation
- Exported types and functions have Go doc comments
- Package-level README.md with usage examples
- RELEASING.md process followed for version tags

### MySQL Version Compliance
- No MySQL 5.x references — CI blocks these
- Use MySQL 8.x compatible syntax and drivers

## Review Process

1. Read the changed files and identify which package(s) are affected
2. Check each focus area above
3. Verify module boundaries are respected
4. Report findings with severity (Critical/High/Medium/Low)
5. Suggest specific fixes

## Output Format

```markdown
## Code Review Results

### Breaking Changes
- [File:Line] Description and migration path

### API Issues
- [File:Line] Description and fix

### Convention Issues
- [File:Line] Description and fix

### Passed Checks
- List of checks that passed
```
