package insrequester

import (
	"github.com/slok/goresilience/errors"
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

	t.Run("it_should_load_retrier_properly", func(t *testing.T) {
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
		assert.ErrorIs(t, err, errors.ErrCircuitOpen)
	})
}
