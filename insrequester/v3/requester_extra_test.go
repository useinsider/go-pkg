package insrequester

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequest_Methods(t *testing.T) {
	tests := []struct {
		name string
		call func(t *testing.T, r Requester, re RequestEntity) (*http.Response, error)
		want string
	}{
		{
			name: "it_should_send_put",
			call: func(t *testing.T, r Requester, re RequestEntity) (*http.Response, error) {
				return r.Put(t.Context(), re)
			},
			want: http.MethodPut,
		},
		{
			name: "it_should_send_delete",
			call: func(t *testing.T, r Requester, re RequestEntity) (*http.Response, error) {
				return r.Delete(t.Context(), re)
			},
			want: http.MethodDelete,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var method string
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method = r.Method
				w.WriteHeader(http.StatusOK)
			}))
			defer ts.Close()

			res, err := tt.call(t, NewRequester(), RequestEntity{Endpoint: ts.URL, Body: []byte(`{}`)})

			require.NoError(t, err)
			assert.Equal(t, http.StatusOK, res.StatusCode)
			assert.Equal(t, tt.want, method)
		})
	}
}

func TestRequest_sendRequestEdgeCases(t *testing.T) {
	t.Run("it_should_return_error_for_unbuildable_request", func(t *testing.T) {
		res, err := NewRequester().Get(t.Context(), RequestEntity{Endpoint: "http://example.com/\x00"})

		assert.Error(t, err)
		assert.Nil(t, res)
	})

	t.Run("it_should_close_partial_response_on_redirect_policy_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/elsewhere", http.StatusFound)
		}))
		defer ts.Close()

		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return errors.New("redirects are forbidden")
			},
		}

		res, err := NewRequester().WithHTTPClient(client).Load().
			Get(t.Context(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "redirects are forbidden")
		assert.Nil(t, res)
	})

	t.Run("it_should_truncate_oversized_error_bodies", func(t *testing.T) {
		big := strings.Repeat("x", 5000)
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(big))
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetriesExhausted)
		assert.Contains(t, err.Error(), "[truncated]")
	})

	t.Run("it_should_report_transport_error_when_circuit_opens_mid_retry", func(t *testing.T) {
		r := NewRequester().
			WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 3}).
			WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen:         1,
				SuccessfulRequiredOnHalfOpen: 1,
				WaitDurationInOpenState:      time.Hour,
			}).Load()

		_, err := r.Get(t.Context(), RequestEntity{Endpoint: "http://127.0.0.1:1"})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrCircuitBreakerOpen)
		assert.Contains(t, err.Error(), "connect", "the transport error should be part of the message")
	})
}

func TestRequest_ConfigDefaultsAndClamps(t *testing.T) {
	t.Run("it_should_default_retry_wait_base", func(t *testing.T) {
		r := NewRequester().WithRetry(RetryConfig{Times: 1})
		require.NotNil(t, r)
		assert.Len(t, r.policies, 1)
	})

	t.Run("it_should_default_circuit_breaker_zero_config", func(t *testing.T) {
		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{})
		require.NotNil(t, r)
		assert.Len(t, r.policies, 1)
	})

	t.Run("it_should_clamp_negative_success_threshold", func(t *testing.T) {
		assert.NotPanics(t, func() {
			NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
				SuccessfulRequiredOnHalfOpen: -3,
			})
		})
	})

	t.Run("it_should_clamp_negative_minimum_request_to_open", func(t *testing.T) {
		assert.NotPanics(t, func() {
			NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
				MinimumRequestToOpen: -5,
			})
		})
	})

	t.Run("it_should_default_rate_based_breaker_thresholding_fields", func(t *testing.T) {
		r := NewRequester().WithCircuitbreaker(CircuitBreakerConfig{
			FailureRateThreshold: 50,
		})
		require.NotNil(t, r)
		assert.Len(t, r.policies, 1)
	})
}
