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

- `insredis` → `go-redis/redis` (v6)
- `inssqs` → `aws-sdk-go-v2` (config, sqs), `smithy-go`, `pkg/errors`
- `insssm` → `Jamil-Najafov/go-aws-ssm` (wraps aws-sdk-go v1)
- `inskinesis` → `aws-sdk-go` v1 (kinesis), `google/uuid`
- `inslogger` → `go.uber.org/zap`
- `insgorm` → `gorm.io/gorm`, `gorm.io/driver/mysql`, `DATA-DOG/go-sqlmock` (test)
- `inssql` → `database/sql` only; `DATA-DOG/go-sqlmock` (test). The caller
  registers the driver.
- `insrequester` (module path `/v2`) → `slok/goresilience` (retry / circuit
  breaker), `pkg/errors`, OpenTelemetry (`go.opentelemetry.io/otel`)
- `insrequester/v3` → `failsafe-go/failsafe-go` (retry / circuit breaker),
  OpenTelemetry
- `inssentry` → `getsentry/sentry-go`
- `inscacheable` → `jellydator/ttlcache/v3`
- Shared: `stretchr/testify`, `golang/mock` (insredis, insrequester v2) /
  `go.uber.org/mock` (inskinesis, inssqs, insrequester v3)

Third-party versions are pinned across modules via `scripts/check-deps.sh`;
update it in the same PR as any version bump.
