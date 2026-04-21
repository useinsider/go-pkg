# Migration Guide: insrequester v2 -> v3

v3 replaces the abandoned `github.com/slok/goresilience` (which transitively pulled the vulnerable `prometheus/client_golang@v0.9.2`, CVE-2022-21698) with the actively maintained `github.com/failsafe-go/failsafe-go`. The public API is unchanged: the same `Requester` interface, the same `NewRequester`, `WithRetry`, `WithCircuitbreaker`, `WithTimeout`, `WithHTTPClient`, `WithHeaders`, `Load`, `RequestEntity`, `Headers`, `RetryConfig`, and `CircuitBreakerConfig` types are all preserved. To migrate, update the import path from `github.com/useinsider/go-pkg/insrequester/v2` to `github.com/useinsider/go-pkg/insrequester/v3` and run `go mod tidy`.

## Behavior changes

- **Retries abort on circuit-open.** When the circuit breaker trips mid-retry, the retry policy now returns `ErrCircuitBreakerOpen` immediately instead of sleeping through the remaining attempts against an already-open breaker.
- **Retries abort on context cancellation / deadline.** v2's `goresilience` retry middleware slept through the full retry budget even after the caller cancelled the context. v3 aborts retries immediately when `ctx.Err()` returns `context.Canceled` or `context.DeadlineExceeded`.
- **Circuit breaker `MinimumRequestToOpen` is now strictly consecutive-failures-in-closed-state.** `goresilience`'s implementation had a latent bug that treated this value as a percent threshold over a rolling window. `failsafe-go`'s `WithFailureThreshold` trips only after `N` consecutive failures and a single intervening success resets the counter. For sliding-window behavior use the new `FailureRateThreshold` / `FailureExecutionThreshold` / `FailureThresholdingPeriod` fields.
- **`ErrTimeout` sentinel is unreachable in v3.** v2 mapped `goresilience`'s internal timeout error to `ErrTimeout`. v3 has no timeout policy; transport-level timeouts surface as the underlying `*url.Error` wrapping `context.DeadlineExceeded`. The `ErrTimeout` symbol is retained (deprecated) for source compatibility.
- **Minimum Go version bumped to 1.25** (v2 required 1.24.0).

## New optional config (additive, zero-valued defaults preserve v2 semantics)

- `RetryConfig.WaitMax` — enables exponential backoff from `WaitBase` up to `WaitMax`. Leave zero for fixed delay.
- `RetryConfig.JitterFactor` — randomizes each delay by `±(delay * factor)` to avoid thundering-herd on downstream recovery. Range `0.0`–`1.0`.
- `CircuitBreakerConfig.FailureRateThreshold` / `FailureExecutionThreshold` / `FailureThresholdingPeriod` — Hystrix-style rate-based tripping (e.g., `50`% errors over a `10s` window with a `20`-sample minimum). When `FailureRateThreshold > 0`, `MinimumRequestToOpen` is ignored. Prefer this for services where concurrent in-flight calls can spuriously trip a count-based breaker.
