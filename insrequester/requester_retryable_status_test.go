package insrequester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_sendRequestTooManyRequests(t *testing.T) {
	t.Run("it_should_exhaust_retries_when_status_is_too_many_requests", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 2}).Load()

		res, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetriesExhausted)
		assert.Nil(t, res)
	})

	t.Run("it_should_retry_when_status_is_too_many_requests", func(t *testing.T) {
		attempts := 0

		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attempts++

			w.WriteHeader(http.StatusTooManyRequests)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 2}).Load()

		_, _ = r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		assert.Equal(t, 3, attempts)
	})

	t.Run("it_should_include_the_too_many_requests_body_in_the_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limited"}`))
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "rate limited")
	})
}
