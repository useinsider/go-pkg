# go-pkg Dependencies

`go-pkg` is a **shared library**, not a runtime service. It has no deployed
pipeline position. Its primary "dependency" relationship is **downstream**:
many Insider backend services import its `ins*` modules, so changes here
propagate outward. Treat backward compatibility of exported APIs as the
binding constraint.

## Downstream consumers

- Consumed broadly across Insider Go services (and Dataforce services in
  particular) via per-module import paths, e.g.
  `github.com/useinsider/go-pkg/insredis`.
- Each module is versioned independently (`<module>/v<version>` tags), so a
  breaking change must be a major bump on that module's path, not a silent
  edit.

## Internal module dependencies

Some modules depend on sibling modules — release the dependency first:

- `inssqs` → `insdash`, `inslogger`
- `insssm` → `inscacheable`

## Notable external dependencies (per module)

- `insredis` → `go-redis/redis`
- `inssqs` → `aws-sdk-go-v2` (config, sqs), `smithy-go`
- `insssm` → `aws-sdk-go-v2` (ssm)
- `inskinesis` → `aws-sdk-go-v2` (kinesis)
- `inslogger` → `go.uber.org/zap`
- `insgorm` / `inssql` → `gorm.io/gorm`, `go-sql-driver/mysql`,
  `DATA-DOG/go-sqlmock` (test)
- `insrequester` → `slok/goresilience` (retry / circuit breaker)
- `inssentry` → `getsentry/sentry-go`
- Shared: `stretchr/testify`, `golang/mock` / `go.uber.org/mock`, `pkg/errors`

Third-party versions are pinned across modules via `scripts/check-deps.sh`;
update it in the same PR as any version bump.
