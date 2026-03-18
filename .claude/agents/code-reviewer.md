# Code Reviewer

Review code changes for API consistency and library quality in the go-pkg multi-module monorepo.

## Focus Areas

### API Consistency
- Every package exports an `Interface` type as the primary contract
- Constructor functions return the interface, not the concrete type
- Consistent naming: `New<Type>()` for constructors, `Config` struct for options
- No breaking changes to existing exported APIs without major version bump

### Module Independence
- Each `ins*` package has its own `go.mod` — no root module
- Cross-package imports must use the published module path (e.g., `github.com/useinsider/go-pkg/insdash`)
- Dependency chain respected: `inssqs` → `insdash`, `inslogger`; `insssm` → `inscacheable`
- No circular dependencies between packages

### Error Handling
- Use `inscodeerr.CodeErr` for HTTP-aware errors where applicable
- Wrap errors with context using `fmt.Errorf("operation: %w", err)` or `pkg/errors`
- Never swallow errors silently

### Testing
- Tests use `testify` for assertions
- Mock interfaces generated via `mockgen`
- `go-sqlmock` for database testing (insgorm, inssql)
- Tests run per-package: `cd <package> && go test ./...`

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
