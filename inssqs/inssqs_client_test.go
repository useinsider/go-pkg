package inssqs

import (
	"net/http"
	"sync"
	"testing"

	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func resetSharedHTTPClient(t *testing.T) {
	t.Helper()

	reset := func() {
		sharedHTTPClientOnce = sync.Once{}
		sharedHTTPClient = nil
	}

	reset()
	t.Cleanup(reset)
}

func TestSharedFrozenHTTPClient(t *testing.T) {
	t.Run("it_should_build_a_client_when_the_sdk_resolved_none", func(t *testing.T) {
		resetSharedHTTPClient(t)

		client := sharedFrozenHTTPClient(nil)

		require.NotNil(t, client)

		_, buildable := client.(*awshttp.BuildableClient)
		assert.False(t, buildable, "shared client must be frozen so the sqs client does not clone it")
	})

	t.Run("it_should_freeze_a_resolved_buildable_client", func(t *testing.T) {
		resetSharedHTTPClient(t)

		client := sharedFrozenHTTPClient(awshttp.NewBuildableClient())

		require.NotNil(t, client)

		_, buildable := client.(*awshttp.BuildableClient)
		assert.False(t, buildable, "shared client must be the frozen client, not the buildable one")
	})

	t.Run("it_should_return_the_first_client_for_a_later_resolved_client", func(t *testing.T) {
		resetSharedHTTPClient(t)

		first := sharedFrozenHTTPClient(awshttp.NewBuildableClient())
		second := sharedFrozenHTTPClient(&countingHTTPClient{inner: http.DefaultClient})

		assert.Same(t, first, second, "a later resolved client must not replace the shared one")
	})
}

func TestNewSQS_sharedHTTPClientAcrossConfigs(t *testing.T) {
	t.Run("it_should_reuse_the_first_calls_connection_for_a_different_config", func(t *testing.T) {
		resetSharedHTTPClient(t)
		setFakeAWSEnv(t)

		ts, remoteAddrs := newRemoteAddrRecordingSQSServer(t)

		NewSQS(Config{Region: "eu-west-1", QueueName: "q1", EndpointUrl: ts.URL})
		NewSQS(Config{Region: "us-east-1", QueueName: "q2", RetryCount: 7, MaxWorkers: 3, EndpointUrl: ts.URL})

		addrs := remoteAddrs()
		require.Len(t, addrs, 2)
		assert.Equal(t, addrs[0], addrs[1], "NewSQS with a different config must reuse the first call's client")
	})
}
