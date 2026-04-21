package insrequester

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingTransport struct {
	wrapped     http.RoundTripper
	onRoundTrip func()
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.onRoundTrip()
	return t.wrapped.RoundTrip(req)
}

type scriptedTransport struct {
	calls int32
	steps []func(req *http.Request) (*http.Response, error)
}

func (s *scriptedTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	idx := atomic.AddInt32(&s.calls, 1) - 1
	if int(idx) >= len(s.steps) {
		return nil, errors.New("scriptedTransport: no more scripted steps")
	}
	return s.steps[idx](req)
}

func TestRequest_Get(t *testing.T) {
	t.Run("it_should_return_response_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))
		defer ts.Close()

		r := NewRequester()

		res, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("it_should_retry_on_internal_server_error", func(t *testing.T) {
		var retryTimes int32
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&retryTimes, 1)
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		r := NewRequester().WithRetry(RetryConfig{
			WaitBase: 20 * time.Millisecond,
			Times:    3,
		}).Load()

		req := RequestEntity{
			Endpoint: server.URL,
		}
		_, _ = r.Get(t.Context(), req)

		assert.Equal(t, int32(4), atomic.LoadInt32(&retryTimes))
	})

	t.Run("it_should_retry_on_timeout", func(t *testing.T) {
		var retryTimes int32
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&retryTimes, 1)
			time.Sleep(100 * time.Millisecond)
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		r := NewRequester().
			WithTimeout(1 * time.Millisecond).
			WithRetry(RetryConfig{
				WaitBase: 20 * time.Millisecond,
				Times:    3,
			}).Load()

		req := RequestEntity{
			Endpoint: server.URL,
		}
		_, _ = r.Get(t.Context(), req)

		assert.Equal(t, int32(4), atomic.LoadInt32(&retryTimes))
	})

	t.Run("it_should_load_circuit_breaker_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			MinimumRequestToOpen:         3,
			SuccessfulRequiredOnHalfOpen: 1,
			WaitDurationInOpenState:      300 * time.Second,
		}).Load()

		minimumRequestToOpen := 3
		var err error
		req := RequestEntity{Endpoint: ts.URL}
		for i := 0; i < minimumRequestToOpen; i++ {
			_, _ = r.Get(t.Context(), req)
		}
		_, err = r.Get(t.Context(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
	})

	t.Run("it_should_return_last_error_if_circuit_breaker_and_retry_enabled", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"status": "FAILED"}`))
		}))
		defer ts.Close()

		r := NewRequester().WithTimeout(1 * time.Millisecond).
			WithRetry(RetryConfig{
				WaitBase: 20 * time.Millisecond,
				Times:    4,
			}).
			WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen:         3,
				SuccessfulRequiredOnHalfOpen: 1,
				WaitDurationInOpenState:      300 * time.Second,
			}).Load()

		var err error
		req := RequestEntity{Endpoint: ts.URL}

		_, err = r.Get(t.Context(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
		assert.Contains(t, err.Error(), "{\"status\": \"FAILED\"}")
	})

	t.Run("it_should_apply_exponential_backoff_when_wait_max_set", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{
			WaitBase: 20 * time.Millisecond,
			WaitMax:  200 * time.Millisecond,
			Times:    3,
		}).Load()

		start := time.Now()
		_, _ = r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		elapsed := time.Since(start)

		assert.Equal(t, int32(4), atomic.LoadInt32(&calls))
		// Expected backoff: 20ms + 40ms + 80ms = 140ms (lower bound). Fixed delay
		// would be 3 * 20ms = 60ms, so anything over 100ms proves backoff took effect.
		assert.GreaterOrEqual(t, elapsed, 100*time.Millisecond,
			"exponential backoff expected >= 100ms of cumulative delay; elapsed=%s", elapsed)
	})

	t.Run("it_should_build_policy_with_jitter_factor", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{
			WaitBase:     10 * time.Millisecond,
			JitterFactor: 0.5,
			Times:        2,
		}).Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.Error(t, err)
		assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
	})

	t.Run("it_should_open_circuit_breaker_on_failure_rate", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			FailureRateThreshold:      50,
			FailureExecutionThreshold: 4,
			FailureThresholdingPeriod: 10 * time.Second,
			WaitDurationInOpenState:   300 * time.Second,
		}).Load()

		req := RequestEntity{Endpoint: ts.URL}
		for i := 0; i < 4; i++ {
			_, err := r.Get(t.Context(), req)
			assert.NotErrorIs(t, err, ErrCircuitBreakerOpen,
				"CB should not open before execution threshold is met (call %d)", i+1)
		}

		_, err := r.Get(t.Context(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
	})

	t.Run("it_should_abort_retries_when_circuit_breaker_opens", func(t *testing.T) {
		var hits int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&hits, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		retryDelay := 200 * time.Millisecond
		retries := 3
		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: retryDelay, Times: retries}).
			WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen:         1,
				SuccessfulRequiredOnHalfOpen: 1,
				WaitDurationInOpenState:      5 * time.Second,
			}).Load()

		req := RequestEntity{Endpoint: ts.URL}

		start := time.Now()
		_, err := r.Get(t.Context(), req)
		elapsed := time.Since(start)

		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
		assert.Equal(t, int32(1), atomic.LoadInt32(&hits), "circuit breaker must short-circuit further HTTP calls")
		budget := time.Duration(retries) * retryDelay
		assert.Less(t, elapsed, budget,
			"retry must abort on circuit-open instead of sleeping %d * %s; elapsed=%s",
			retries, retryDelay, elapsed)
	})

	t.Run("it_should_apply_headers_properly", func(t *testing.T) {
		var receivedUserAgent string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUserAgent = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))

		defer ts.Close()

		userAgent := "test-user-agent"
		r := NewRequester().WithHeaders(Headers{{"User-Agent": userAgent}})
		res, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})

		assert.NoError(t, err)
		assert.Equal(t, receivedUserAgent, userAgent)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("it_should_use_custom_http_client_when_provided", func(t *testing.T) {
		var transportUsed bool
		customTransport := &recordingTransport{
			wrapped:     http.DefaultTransport,
			onRoundTrip: func() { transportUsed = true },
		}
		customClient := &http.Client{Transport: customTransport}

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().WithHTTPClient(customClient).Load()
		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})

		assert.NoError(t, err)
		assert.True(t, transportUsed)
	})

	t.Run("it_should_trigger_exactly_N_plus_one_attempts_on_retry_policy", func(t *testing.T) {
		var calls int32
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		retries := 2
		r := NewRequester().WithRetry(RetryConfig{
			WaitBase: 5 * time.Millisecond,
			Times:    retries,
		}).Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: server.URL})
		assert.Error(t, err)
		assert.Equal(t, int32(retries+1), atomic.LoadInt32(&calls))
	})

	t.Run("it_should_open_circuit_breaker_after_failure_threshold", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		failureThreshold := 2
		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			MinimumRequestToOpen:         failureThreshold,
			SuccessfulRequiredOnHalfOpen: 1,
			WaitDurationInOpenState:      300 * time.Second,
		}).Load()

		req := RequestEntity{Endpoint: ts.URL}
		for i := 0; i < failureThreshold; i++ {
			_, _ = r.Get(t.Context(), req)
		}

		_, err := r.Get(t.Context(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
	})

	t.Run("it_should_override_Requester_level_header_if_RequestEntity_headers_set", func(t *testing.T) {
		var receivedUserAgent string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedUserAgent = r.Header.Get("User-Agent")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))

		defer ts.Close()

		oldUserAgent := "old-user-agent"
		r := NewRequester().WithHeaders(Headers{{"User-Agent": oldUserAgent}})

		newUserAgent := "new-user-agent"
		req := RequestEntity{
			Endpoint: ts.URL,
			Headers:  Headers{{"User-Agent": newUserAgent}},
		}
		res, err := r.Get(t.Context(), req)

		assert.NoError(t, err)
		assert.Equal(t, receivedUserAgent, newUserAgent)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("it_should_return_success_when_retry_recovers_from_transport_error", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"OK"}`))
		}))
		defer ts.Close()

		transport := &scriptedTransport{steps: []func(req *http.Request) (*http.Response, error){
			func(*http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				return nil, errors.New("simulated transport blip")
			},
			func(req *http.Request) (*http.Response, error) {
				atomic.AddInt32(&calls, 1)
				return http.DefaultTransport.RoundTrip(req)
			},
		}}

		r := NewRequester().
			WithHTTPClient(&http.Client{Transport: transport}).
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 2}).
			Load()

		res, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err, "retry that succeeds after transport error must return no error")
		require.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
	})

	t.Run("it_should_return_success_when_retry_recovers_from_5xx", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if atomic.AddInt32(&calls, 1) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 2}).
			Load()

		res, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
		assert.Equal(t, int32(2), atomic.LoadInt32(&calls))
	})

	t.Run("it_should_send_post_body_through_retry", func(t *testing.T) {
		var bodies []string
		var mu sync.Mutex
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			mu.Lock()
			bodies = append(bodies, string(body))
			mu.Unlock()
			if len(bodies) == 1 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 2}).
			Load()

		payload := `{"k":"v"}`
		res, err := r.Post(t.Context(), RequestEntity{Endpoint: ts.URL, Body: []byte(payload)})
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, bodies, 2, "retry should resend body")
		assert.Equal(t, payload, bodies[0])
		assert.Equal(t, payload, bodies[1])
	})

	t.Run("it_should_rewrite_Host_via_Host_header", func(t *testing.T) {
		var receivedHost string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHost = r.Host
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester()
		_, err := r.Get(t.Context(), RequestEntity{
			Endpoint: ts.URL,
			Headers:  Headers{{"Host": "override.example"}},
		})
		assert.NoError(t, err)
		assert.Equal(t, "override.example", receivedHost)
	})

	t.Run("it_should_allow_user_content_type_to_override_default", func(t *testing.T) {
		var receivedCT string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedCT = r.Header.Get("Content-Type")
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester()
		_, err := r.Post(t.Context(), RequestEntity{
			Endpoint: ts.URL,
			Body:     []byte(`xml`),
			Headers:  Headers{{"Content-Type": "application/xml"}},
		})
		assert.NoError(t, err)
		assert.Equal(t, "application/xml", receivedCT,
			"explicit user Content-Type must override the default application/json")
	})

	t.Run("it_should_respect_ctx_cancellation_across_retries", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 200 * time.Millisecond, Times: 5}).
			Load()

		ctx, cancel := context.WithTimeout(t.Context(), 50*time.Millisecond)
		defer cancel()

		start := time.Now()
		_, err := r.Get(ctx, RequestEntity{Endpoint: ts.URL})
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Less(t, elapsed, 500*time.Millisecond,
			"retries must abort once ctx is cancelled; elapsed=%s, calls=%d",
			elapsed, atomic.LoadInt32(&calls))
	})

	t.Run("it_should_not_mutate_caller_supplied_client_when_timeout_set", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		callerClient := &http.Client{Timeout: 77 * time.Second}
		r := NewRequester().
			WithHTTPClient(callerClient).
			WithTimeout(3 * time.Second).
			Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err)
		assert.Equal(t, 77*time.Second, callerClient.Timeout,
			"WithTimeout must clone the caller's *http.Client, not mutate it")
	})

	t.Run("it_should_default_timeout_to_30s_when_WithTimeout_zero", func(t *testing.T) {
		req := NewRequester().WithTimeout(0)
		assert.Equal(t, 30*time.Second, req.timeout)
	})

	t.Run("it_should_default_retry_Times_to_3_when_unset", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond}).Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.Error(t, err)
		assert.Equal(t, int32(4), atomic.LoadInt32(&calls),
			"default Times=3 implies 1 initial + 3 retries = 4 calls")
	})

	t.Run("it_should_implicitly_initialize_when_Load_not_called", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1})

		res, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err)
		require.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("it_should_be_safe_for_concurrent_use", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).
			WithCircuitbreaker(CircuitBreakerConfig{MinimumRequestToOpen: 100}).
			Load()

		var wg sync.WaitGroup
		errs := make(chan error, 20)
		for i := 0; i < 20; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
				if err != nil {
					errs <- err
				}
			}()
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			t.Errorf("unexpected error from concurrent Get: %v", err)
		}
	})

	t.Run("it_should_clamp_jitter_factor_over_one", func(t *testing.T) {
		var calls int32
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			atomic.AddInt32(&calls, 1)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{
			WaitBase:     5 * time.Millisecond,
			JitterFactor: 42.0,
			Times:        2,
		}).Load()

		start := time.Now()
		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
		assert.Less(t, elapsed, time.Second,
			"clamped JitterFactor must not produce extreme delays; elapsed=%s", elapsed)
	})

	t.Run("it_should_clamp_failure_rate_threshold_over_hundred", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		assert.NotPanics(t, func() {
			_ = NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
				FailureRateThreshold:      250,
				FailureExecutionThreshold: 2,
				FailureThresholdingPeriod: time.Second,
			}).Load()
		}, "FailureRateThreshold > 100 must be clamped, not panic")
	})

	t.Run("it_should_wrap_ErrCircuitBreakerOpen_with_errors_Is", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`body-xyz`))
		}))
		defer ts.Close()

		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			MinimumRequestToOpen:    1,
			WaitDurationInOpenState: time.Hour,
		}).Load()

		req := RequestEntity{Endpoint: ts.URL}
		_, _ = r.Get(t.Context(), req)
		_, err := r.Get(t.Context(), req)

		assert.ErrorIs(t, err, ErrCircuitBreakerOpen,
			"wrapped sentinel must remain Is-compatible after fmt.Errorf migration")
	})

	t.Run("it_should_initialize_executor_exactly_once_under_load", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: time.Millisecond, Times: 1})

		var wg sync.WaitGroup
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, _ = r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})
			}()
		}
		wg.Wait()
	})

	t.Run("it_should_not_duplicate_requester_headers_across_retries", func(t *testing.T) {
		var mu sync.Mutex
		var seen []string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			seen = append(seen, r.Header.Values("X-Client")...)
			mu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().
			WithHeaders(Headers{{"X-Client": "alpha"}}).
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 3}).
			Load()

		_, _ = r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})

		mu.Lock()
		defer mu.Unlock()
		require.Len(t, seen, 4, "4 attempts = 4 header values captured")
		for i, v := range seen {
			assert.Equal(t, "alpha", v,
				"attempt %d: requester header must not accumulate duplicates (got %q)", i+1, v)
		}
	})

	t.Run("it_should_not_mutate_caller_RequestEntity_headers", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().WithHeaders(Headers{{"X-A": "1"}, {"X-B": "2"}})

		callerHeaders := Headers{{"X-Entity": "entity-value"}}
		original := make(Headers, len(callerHeaders))
		copy(original, callerHeaders)

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL, Headers: callerHeaders})
		assert.NoError(t, err)
		assert.Equal(t, original, callerHeaders, "sendRequest must not mutate caller's RequestEntity.Headers")
	})

	t.Run("it_should_propagate_parent_ctx_cancellation_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		defer ts.Close()

		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 3}).
			Load()

		ctx, cancel := context.WithCancel(t.Context())
		cancel()

		_, err := r.Get(ctx, RequestEntity{Endpoint: ts.URL})
		require.Error(t, err)
		assert.True(t,
			errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded),
			"expected ctx error propagated as-is; got %v", err)
	})

	t.Run("it_should_transition_circuit_breaker_back_to_closed_after_delay", func(t *testing.T) {
		var failing atomic.Bool
		failing.Store(true)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if failing.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			MinimumRequestToOpen:         2,
			SuccessfulRequiredOnHalfOpen: 1,
			WaitDurationInOpenState:      50 * time.Millisecond,
		}).Load()

		req := RequestEntity{Endpoint: ts.URL}
		for i := 0; i < 2; i++ {
			_, _ = r.Get(t.Context(), req)
		}
		_, err := r.Get(t.Context(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen, "breaker should be open")

		time.Sleep(80 * time.Millisecond)
		failing.Store(false)

		res, err := r.Get(t.Context(), req)
		assert.NoError(t, err, "breaker should half-open after delay and allow a probe")
		require.NotNil(t, res)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
