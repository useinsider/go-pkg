package insrequester

import "errors"

var (
	ErrCircuitBreakerOpen = errors.New("circuit breaker is open")
	ErrTimeout            = errors.New("timeout")

	ErrRetryable   = errors.New("retryable error")
	ErrReadingBody = errors.New("error reading body")
)
