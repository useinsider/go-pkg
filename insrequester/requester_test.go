package insrequester

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/useinsider/go-pkg/insredis"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequest_Get(t *testing.T) {
	t.Run("it_should_return_error_when_request_is_nil", func(t *testing.T) {
		ctx := context.Background()
		type x string

		var g x
		g = "val"

		ctx = context.WithValue(ctx, g, "rafet")
		fmt.Printf("%v", ctx.Value("val"))

	})

	t.Run("it_should_return_response_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status": "OK"}`))
		}))
		defer ts.Close()

		r := NewRequester(insredis.Init("localhost:6379", 101))

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

		r := NewRequester(insredis.Init("localhost:6379", 101)).WithRetry(RetryConfig{
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
		redos := insredis.Init("localhost:6379", 101)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester(redos).
			SetName(ts.URL).
			WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen:         3,
				SuccessfulRequiredOnHalfOpen: 1,
				WaitDurationInOpenState:      300 * time.Second,
			}).OnCircularBreakerOpen(func() {
			fmt.Println("circuit breaker is open")
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

	//write a case for timeout
	t.Run("it_should_load_timeout_properly", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(2 * time.Second)
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester(insredis.Init("localhost:6379", 101)).WithTimeout(1).Load()
		req := RequestEntity{Endpoint: ts.URL}
		_, err := r.Get(req)
		assert.ErrorIs(t, err, ErrTimeout)
	})
}
