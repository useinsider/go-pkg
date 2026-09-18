package inssqs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRemoteAddrRecordingSQSServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()

	var (
		mu    sync.Mutex
		addrs []string
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.ReadAll(r.Body)

		mu.Lock()
		addrs = append(addrs, r.RemoteAddr)
		mu.Unlock()

		w.Header().Set("Content-Type", "application/x-amz-json-1.0")
		_ = json.NewEncoder(w).Encode(map[string]string{"QueueUrl": "https://sqs.test/queue"})
	}))
	t.Cleanup(ts.Close)

	return ts, func() []string {
		mu.Lock()
		defer mu.Unlock()

		return append([]string(nil), addrs...)
	}
}

type countingHTTPClient struct {
	inner HTTPClient
	calls atomic.Int32
}

func (c *countingHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.calls.Add(1)

	return c.inner.Do(req)
}

func TestNewSQS_httpClient(t *testing.T) {
	t.Run("it_should_reuse_http_connection_across_NewSQS_calls", func(t *testing.T) {
		setFakeAWSEnv(t)
		ts, remoteAddrs := newRemoteAddrRecordingSQSServer(t)

		for i := 0; i < 3; i++ {
			NewSQS(Config{Region: "eu-west-1", QueueName: "q", EndpointUrl: ts.URL})
		}

		addrs := remoteAddrs()
		require.Len(t, addrs, 3)
		assert.Equal(t, addrs[0], addrs[1], "second NewSQS must reuse the first call's keep-alive connection")
		assert.Equal(t, addrs[0], addrs[2], "third NewSQS must reuse the first call's keep-alive connection")
	})

	t.Run("it_should_use_injected_http_client", func(t *testing.T) {
		setFakeAWSEnv(t)
		ts := newFakeSQSServer(t, http.StatusOK)
		client := &countingHTTPClient{inner: http.DefaultClient}

		NewSQS(Config{Region: "eu-west-1", QueueName: "q", EndpointUrl: ts.URL, HTTPClient: client})

		assert.Equal(t, int32(1), client.calls.Load(), "GetQueueUrl must go through the injected client")
	})
}
