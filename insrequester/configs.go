package insrequester

import "time"

type RequesterConfig struct {
	MaxIdleConns        int
	MaxConnsPerHost     int
	MaxIdleConnsPerHost int
}

type CircuitBreakerConfig struct {
	MinimumRequestToOpen         int
	SuccessfulRequiredOnHalfOpen int
	WaitDurationInOpenState      time.Duration
}

type RetryConfig struct {
	WaitBase time.Duration
	Times    int
}
