# go-pkg

> ## BREAKING CHANGES (v1.0.0)
>
> **Effective after v0.12.0** - This repository has been converted to a **multi-module structure**.
>
> ### What changed:
> - The root `go.mod` has been removed
> - Each package now has its own `go.mod` with independent version tags (e.g., `insredis/v1.0.0`)
>
> ### inssql / insgorm split:
> The old `inssql` package has been split into two separate packages:
>
> | Old Import | New Import | Description |
> |------------|------------|-------------|
> | `go-pkg/inssql` (SQL functions) | `go-pkg/inssql` | Pure `database/sql` - `Init()`, `New()`, `GetClient()`, `MockSql()` |
> | `go-pkg/inssql` (GORM functions) | `go-pkg/insgorm` | GORM wrapper - `WrapWithGorm()`, `NewGorm()`, `GetGormClient()`, `MockGorm()` |
>
> ### Migration:
> ```bash
> # Update your go.mod from:
> require github.com/useinsider/go-pkg v0.12.0
>
> # To specific packages:
> require github.com/useinsider/go-pkg/insredis v1.0.0
> require github.com/useinsider/go-pkg/inssql v1.0.0
> require github.com/useinsider/go-pkg/insgorm v1.0.0  # if using GORM functions
> ```

## Multi-Module Repository

This repository uses a multi-module structure. Each package is independently versioned and can be imported separately.

## Available Packages

| Package | Import Path | Description |
|---------|-------------|-------------|
| inscacheable | `github.com/useinsider/go-pkg/inscacheable` | TTL cache wrapper |
| inscodeerr | `github.com/useinsider/go-pkg/inscodeerr` | HTTP error codes |
| insdash | `github.com/useinsider/go-pkg/insdash` | Utility functions |
| insgorm | `github.com/useinsider/go-pkg/insgorm` | GORM wrapper |
| inskinesis | `github.com/useinsider/go-pkg/inskinesis` | AWS Kinesis client |
| inslogger | `github.com/useinsider/go-pkg/inslogger` | Zap logger wrapper |
| insredis | `github.com/useinsider/go-pkg/insredis` | Redis client |
| insrequester | `github.com/useinsider/go-pkg/insrequester` | HTTP client with retry/circuit breaker |
| inssentry | `github.com/useinsider/go-pkg/inssentry` | Sentry integration |
| inssimpleroute | `github.com/useinsider/go-pkg/inssimpleroute` | Simple HTTP router |
| inssql | `github.com/useinsider/go-pkg/inssql` | SQL client |
| inssqs | `github.com/useinsider/go-pkg/inssqs` | AWS SQS client |
| insssm | `github.com/useinsider/go-pkg/insssm` | AWS SSM parameter store |

## Installation

Install only the packages you need:

```bash
go get github.com/useinsider/go-pkg/insredis@v1.0.0
go get github.com/useinsider/go-pkg/inslogger@v1.0.0
```

```go
package main

import (
    "github.com/useinsider/go-pkg/insredis"
    "github.com/useinsider/go-pkg/inslogger"
)
```

## Migration from Single Module

If you were using the old single-module version (`github.com/useinsider/go-pkg`), update your imports to use specific package versions:

```go
// Before
import "github.com/useinsider/go-pkg/insredis"

// After - same import, but go.mod changes:
require github.com/useinsider/go-pkg/insredis v1.0.0
```

## Test

Run tests for a specific package:

```bash
cd insredis && go test ./... -count=1
```
