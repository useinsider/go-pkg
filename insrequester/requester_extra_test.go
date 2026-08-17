package insrequester

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newMethodEchoServer(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var methods []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(ts.Close)
	return ts, &methods
}

func TestRequest_Methods(t *testing.T) {
	tests := []struct {
		name string
		call func(r Requester, re RequestEntity) (*http.Response, error)
		want string
	}{
		{
			name: "it_should_send_post",
			call: func(r Requester, re RequestEntity) (*http.Response, error) {
				return r.Post(context.Background(), re)
			},
			want: http.MethodPost,
		},
		{
			name: "it_should_send_put",
			call: func(r Requester, re RequestEntity) (*http.Response, error) {
				return r.Put(context.Background(), re)
			},
			want: http.MethodPut,
		},
		{
			name: "it_should_send_delete",
			call: func(r Requester, re RequestEntity) (*http.Response, error) {
				return r.Delete(context.Background(), re)
			},
			want: http.MethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, methods := newMethodEchoServer(t)

			res, err := tt.call(NewRequester(), RequestEntity{Endpoint: ts.URL, Body: []byte(`{}`)})

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			require.Len(t, *methods, 1)
			assert.Equal(t, tt.want, (*methods)[0])
		})
	}
}

func TestRequest_sendRequestEdgeCases(t *testing.T) {
	t.Run("it_should_return_error_for_unbuildable_request", func(t *testing.T) {
		r := NewRequester()

		res, err := r.Get(context.Background(), RequestEntity{Endpoint: "http://example.com/\x00"})

		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("it_should_clone_custom_client_when_timeout_set", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		callerClient := &http.Client{Timeout: 77 * time.Second}
		r := NewRequester().WithHTTPClient(callerClient).WithTimeout(3 * time.Second).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		assert.NoError(t, err)
		assert.Equal(t, 77*time.Second, callerClient.Timeout,
			"WithTimeout must clone the caller's client, not mutate it")
	})

	t.Run("it_should_truncate_oversized_error_bodies", func(t *testing.T) {
		big := strings.Repeat("x", 5000)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(big))
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetriesExhausted)
		assert.Contains(t, err.Error(), "[truncated]")
	})

	t.Run("it_should_report_transport_error_when_circuit_opens_mid_retry", func(t *testing.T) {
		// Unreachable endpoint: every attempt fails at the transport level, so
		// when the breaker opens during the retry loop the last transport
		// error is wrapped around ErrCircuitBreakerOpen.
		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 5}).
			WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen:         2,
				SuccessfulRequiredOnHalfOpen: 1,
				WaitDurationInOpenState:      time.Hour,
			}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: "http://127.0.0.1:1"})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
		assert.Contains(t, err.Error(), "connect", "the transport error should be part of the message")
	})

	t.Run("it_should_apply_host_header_to_request_host", func(t *testing.T) {
		var receivedHost string
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			receivedHost = r.Host
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		_, err := NewRequester().Get(context.Background(), RequestEntity{
			Endpoint: ts.URL,
			Headers:  Headers{{"Host": "override.example"}},
		})

		assert.NoError(t, err)
		assert.Equal(t, "override.example", receivedHost)
	})
}

func TestRequest_ConfigDefaults(t *testing.T) {
	t.Run("it_should_default_retry_config_zero_values", func(t *testing.T) {
		r := NewRequester().WithRetry(RetryConfig{})
		require.NotNil(t, r)
		assert.Len(t, r.middlewares, 1)
	})

	t.Run("it_should_default_circuit_breaker_config_zero_values", func(t *testing.T) {
		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{})
		require.NotNil(t, r)
		assert.Len(t, r.middlewares, 1)
	})

	t.Run("it_should_default_timeout_to_30s_when_zero", func(t *testing.T) {
		r := NewRequester().WithTimeout(0)
		assert.Equal(t, 30*time.Second, r.timeout)
	})

	t.Run("it_should_keep_explicit_timeout", func(t *testing.T) {
		r := NewRequester().WithTimeout(5 * time.Second)
		assert.Equal(t, 5*time.Second, r.timeout)
	})
}
