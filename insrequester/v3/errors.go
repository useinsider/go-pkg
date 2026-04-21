package insrequester

import "errors"

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")

	// Deprecated: v3 does not configure a timeout policy, so this sentinel is
	// never returned. It is retained for backward compatibility with v2 callers
	// that referenced the symbol. Clients waiting on `errors.Is(err, ErrTimeout)`
	// will observe the transport-level error directly instead.
	ErrTimeout = errors.New("timeout")

	ErrRetryable        = errors.New("retryable error")
	ErrRetriesExhausted = errors.New("retries exhausted")
	ErrReadingBody      = errors.New("error reading body")
)
