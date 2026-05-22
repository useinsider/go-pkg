# Go code style — go-pkg

Scoped to this shared library. Idioms here are chosen so every `ins*` package
stays interchangeable for callers.

## Formatting

- `gofmt -s` (simplify) is the only allowed formatter. No other opinions.
- `goimports` grouping: stdlib, then third-party, then `github.com/useinsider/...`.
- Line length: no hard limit; break when a line hurts readability, not before.

## Naming

- Package names are always `ins<thing>` — e.g. `insredis`, `insgorm`. Never
  stutter in exported names (`insredis.Client`, not `insredis.RedisClient`).
- Constructors: `New(cfg Config) Interface`. Never return the concrete struct.
- Config struct is always `Config` (one per package). Optional overrides go
  through additional constructors (`NewWithFoo`) or functional options, not
  new exported fields on existing callers' zero-values.
- Errors are exported as `Err<Reason>` sentinels (e.g. `ErrNotFound`) when
  callers should compare with `errors.Is`; otherwise wrap with
  `fmt.Errorf("...: %w", err)`.

## Structure

- One interface per package, named `Interface`. Callers depend on it, not on
  the concrete type.
- Unexported struct `client` implements `Interface`.
- Mock lives in `<pkg>_mock.go` alongside the real file, generated with
  `mockgen`. Commit the generated file — CI does not regenerate.

## Things to avoid

- `panic` in library code. Return errors; the caller decides whether to panic.
- `init()` functions with side effects (registering metrics, dialing network,
  reading env). Make it explicit through `New`.
- Global mutable state. Every package is used by many services concurrently.
- Direct `os.Getenv` reads. Config comes through `Config`, caller loads env.
