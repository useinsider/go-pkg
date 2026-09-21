---
name: add-package
description: Scaffold a new ins* package in the go-pkg multi-module monorepo
disable-model-invocation: true
---

Create a new package: $ARGUMENTS

## Steps

1. **Create package directory**
   - Name: `ins<name>/` (e.g., `insmetrics/`)
   - All package names are prefixed with `ins`

2. **Initialize Go module**
   ```bash
   cd ins<name>
   go mod init github.com/useinsider/go-pkg/ins<name>
   ```
   - Set Go version to match other recent packages (`insrequester/v3/go.mod` currently has the highest floor; the `/v2` module lives in `insrequester/go.mod`)

3. **Create main package file** (`ins<name>/<name>.go`)
   - Define the primary `Interface` type
   - Create `Config` struct for initialization options
   - Implement `New()` constructor that returns the interface
   - Example pattern (reference `insredis/`, `inslogger/`, `inssqs/`):
     ```go
     package ins<name>

     type Interface interface {
         // Methods
     }

     type Config struct {
         // Options
     }

     type client struct {
         config Config
     }

     func New(cfg Config) Interface {
         return &client{config: cfg}
     }
     ```

4. **Create test file** (`ins<name>/<name>_test.go`)
   - Use `testify` for assertions
   - Table-driven tests with `t.Run()` subtests

5. **Create README.md** with:
   - Package description
   - Installation: `go get github.com/useinsider/go-pkg/ins<name>`
   - Usage example
   - Configuration options

6. **If this package depends on other ins* packages**:
   - Add to CLAUDE.md dependency chain
   - Document release order in RELEASING.md

7. **Update repository documentation**
   - Add entry to root `README.md` package table
   - Add entry to `CLAUDE.md` repository structure
   - Update `scripts/check-deps.sh` if needed

8. **CI needs no edit**
   - `.github/workflows/lint.yml` and `scripts/coverage.sh` (the `Unit Tests`
     check) both discover modules with `find . -name go.mod`, so the new
     module is linted and tested automatically. Do not add a matrix entry
     or a new job — the `golangci-lint`, `Unit Tests` and `Integration Tests`
     check names are branch-protection contexts and must stay as they are.
     The one exception is a module whose tests need a real dependency:
     `test/integration` has its own job for exactly that reason.

9. **Verify**
   ```bash
   cd ins<name>
   go mod tidy
   go test -race ./...
   go vet ./...
   golangci-lint run --config ../.golangci.yaml ./...   # must print "0 issues."
   ```
