package insrequester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type recordingTransport struct {
	wrapped     http.RoundTripper
	onRoundTrip func()
}

func (t *recordingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	t.onRoundTrip()
	return t.wrapped.RoundTrip(req)
}

func TestRequest_Get(t *testing.T) {
	t.Run("it_should_return_response_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))
		defer ts.Close()

		r := NewRequester()

		res, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})
		assert.NoError(t, err)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})

	t.Run("it_should_retry_on_internal_server_error", func(t *testing.T) {
		retryTimes := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retryTimes++
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
		_, _ = r.Get(context.Background(), req)

		assert.Equal(t, 4, retryTimes)
	})

	t.Run("it_should_retry_on_timeout", func(t *testing.T) {
		retryTimes := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			retryTimes++
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
		_, _ = r.Get(context.Background(), req)

		assert.Equal(t, 4, retryTimes)
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
			_, _ = r.Get(context.Background(), req)
		}
		_, err = r.Get(context.Background(), req)
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

		_, err = r.Get(context.Background(), req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
		assert.Contains(t, err.Error(), "{\"status\": \"FAILED\"}")
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
		res, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

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
		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		assert.NoError(t, err)
		assert.True(t, transportUsed)
	})

	t.Run("it_should_trigger_exactly_N_plus_one_attempts_on_retry_policy", func(t *testing.T) {
		calls := 0
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls++
			w.WriteHeader(http.StatusInternalServerError)
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		retries := 2
		r := NewRequester().WithRetry(RetryConfig{
			WaitBase: 5 * time.Millisecond,
			Times:    retries,
		}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: server.URL})
		assert.Error(t, err)
		assert.Equal(t, retries+1, calls)
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
			_, _ = r.Get(context.Background(), req)
		}

		_, err := r.Get(context.Background(), req)
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
		res, err := r.Get(context.Background(), req)

		assert.NoError(t, err)
		assert.Equal(t, receivedUserAgent, newUserAgent)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
