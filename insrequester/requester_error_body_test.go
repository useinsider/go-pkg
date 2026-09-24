package insrequester

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_errorBodySummaryEmptyBody(t *testing.T) {
	tests := []struct {
		name   string
		status string
		code   int
	}{
		{
			name:   "it_should_return_the_bare_status_when_internal_server_error_has_no_body",
			status: "500 Internal Server Error",
			code:   http.StatusInternalServerError,
		},
		{
			name:   "it_should_return_the_bare_status_when_too_many_requests_has_no_body",
			status: "429 Too Many Requests",
			code:   http.StatusTooManyRequests,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := &http.Response{
				Status:     tt.status,
				StatusCode: tt.code,
				Body:       io.NopCloser(strings.NewReader("")),
			}

			assert.Equal(t, tt.status, errorBodySummary(res))
		})
	}

	t.Run("it_should_not_append_a_separator_when_the_body_is_empty", func(t *testing.T) {
		res := &http.Response{
			Status:     "500 Internal Server Error",
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("")),
		}

		assert.NotContains(t, errorBodySummary(res), " : ")
	})
}

func TestRequest_sendRequestEmptyErrorBody(t *testing.T) {
	t.Run("it_should_carry_the_bare_status_into_the_retries_exhausted_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.ErrorIs(t, err, ErrRetriesExhausted)
		assert.Contains(t, err.Error(), "500 Internal Server Error")
	})

	t.Run("it_should_not_include_an_empty_body_separator_in_the_retries_exhausted_error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		r := NewRequester().WithRetry(RetryConfig{WaitBase: 5 * time.Millisecond, Times: 1}).Load()

		_, err := r.Get(context.Background(), RequestEntity{Endpoint: ts.URL})

		require.Error(t, err)
		assert.NotContains(t, err.Error(), " : ")
	})
}
