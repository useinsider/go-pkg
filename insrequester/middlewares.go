package insrequester

import (
	"fmt"
	"github.com/slok/goresilience/circuitbreaker"
	"github.com/slok/goresilience/retry"
	"github.com/slok/goresilience/timeout"
	"sync"
	"time"
)

func (r *Request) WithRetry(config RetryConfig) *Request {
	if config.WaitBase == 0 {
		config.WaitBase = 200 * time.Millisecond
	}

	if config.Times == 0 {
		config.Times = 3
	}

	mw := retry.NewMiddleware(retry.Config{
		WaitBase: config.WaitBase,
		Times:    config.Times,
	})

	r.middlewares = append(r.middlewares, mw)

	return r
}
func (r *Request) WithCircuitbreaker(config CircuitBreakerConfig) *Request {
	if config.MinimumRequestToOpen == 0 {
		config.MinimumRequestToOpen = 3
	}

	if config.SuccessfulRequiredOnHalfOpen == 0 {
		config.SuccessfulRequiredOnHalfOpen = 1
	}

	if config.WaitDurationInOpenState == 0 {
		config.WaitDurationInOpenState = 5 * time.Second
	}

	rec := newCBRecorder(r)

	mw := circuitbreaker.NewMiddleware(circuitbreaker.Config{
		MinimumRequestToOpen:         config.MinimumRequestToOpen,
		SuccessfulRequiredOnHalfOpen: config.SuccessfulRequiredOnHalfOpen,
		WaitDurationInOpenState:      config.WaitDurationInOpenState,
	}, &rec)

	r.middlewares = append(r.middlewares, mw)

	return r
}

func (r *Request) WithTimeout(timeoutSeconds int) *Request {
	if timeoutSeconds == 0 {
		r.timeout = 30
	} else {
		r.timeout = timeoutSeconds
	}

	mw := timeout.NewMiddleware(timeout.Config{
		Timeout: time.Duration(r.timeout) * time.Second,
	})
	r.middlewares = append(r.middlewares, mw)

	return r
}

type bw struct {
	requester          *Request
	nextIndexToReplace int
	mu                 sync.Mutex
}

func newCBRecorder(r *Request) circuitbreaker.Recorder {
	b := &bw{
		requester: r,
	}

	return b
}

func (b *bw) Inc(err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.requester.cache.Incr(b.cbTotalRedisKey())
	if err != nil {
		b.requester.cache.Incr(b.cbErrRedisKey())
	}
}

func (b *bw) Reset() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.requester.cache.Set(b.cbTotalRedisKey(), 0, 0)
	b.requester.cache.Set(b.cbErrRedisKey(), 0, 0)
}

func (b *bw) ErrorRate() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	total, _ := b.requester.cache.Get(b.cbTotalRedisKey()).Float64()
	errs, _ := b.requester.cache.Get(b.cbErrRedisKey()).Float64()

	return errs / total
}

func (b *bw) TotalRequests() float64 {
	b.mu.Lock()
	defer b.mu.Unlock()

	total, _ := b.requester.cache.Get(b.cbTotalRedisKey()).Float64()

	return total
}

func (b *bw) cbTotalRedisKey() string {
	return fmt.Sprintf("insrequester:%s:total", b.requester.Name())
}

func (b *bw) cbErrRedisKey() string {
	return fmt.Sprintf("insrequester:%s:err", b.requester.Name())
}
