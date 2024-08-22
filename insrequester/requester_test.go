package insrequester

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequest_Get(t *testing.T) {
	t.Run("it_should_return_response_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))
		defer ts.Close()

		r := NewRequester()

		res, err := r.Get(RequestEntity{Endpoint: ts.URL})
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
		_, _ = r.Get(req)

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
		_, _ = r.Get(req)

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
			_, _ = r.Get(req)
		}
		_, err = r.Get(req)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
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
		res, err := r.Get(RequestEntity{Endpoint: ts.URL})

		assert.NoError(t, err)
		assert.Equal(t, receivedUserAgent, userAgent)
		assert.Equal(t, http.StatusOK, res.StatusCode)
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
		res, err := r.Get(req)

		assert.NoError(t, err)
		assert.Equal(t, receivedUserAgent, newUserAgent)
		assert.Equal(t, http.StatusOK, res.StatusCode)
	})
}
