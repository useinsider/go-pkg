package sqs

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newProxyUnderTest(t *testing.T) API {
	t.Helper()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")

		switch {
		case strings.HasSuffix(target, "GetQueueUrl"):
			_ = json.NewEncoder(w).Encode(map[string]string{"QueueUrl": "https://sqs.test/queue"})
		case strings.HasSuffix(target, "SendMessageBatch"):
			_, _ = w.Write([]byte(`{"Successful":[{"Id":"id-1","MessageId":"m-1","MD5OfMessageBody":"841a2d689ad86bd1611447453c22c6fc"}],"Failed":[]}`))
		case strings.HasSuffix(target, "DeleteMessageBatch"):
			_, _ = w.Write([]byte(`{"Successful":[{"Id":"id-1"}],"Failed":[]}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(ts.Close)

	client := awssqs.New(awssqs.Options{
		Region:       "eu-west-1",
		BaseEndpoint: aws.String(ts.URL),
		Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
	})

	return NewSQSProxy(client)
}

func TestNewSQSProxy(t *testing.T) {
	t.Run("it_should_wrap_the_client", func(t *testing.T) {
		p := newProxyUnderTest(t)
		assert.NotNil(t, p)
	})
}

func TestProxy_GetQueueUrl(t *testing.T) {
	t.Run("it_should_forward_the_call", func(t *testing.T) {
		p := newProxyUnderTest(t)

		out, err := p.GetQueueUrl(context.Background(), &awssqs.GetQueueUrlInput{
			QueueName: aws.String("q"),
		})

		require.NoError(t, err)
		assert.Equal(t, "https://sqs.test/queue", aws.ToString(out.QueueUrl))
	})
}

func TestProxy_SendMessageBatch(t *testing.T) {
	t.Run("it_should_forward_the_call", func(t *testing.T) {
		p := newProxyUnderTest(t)

		out, err := p.SendMessageBatch(context.Background(), &awssqs.SendMessageBatchInput{
			QueueUrl: aws.String("https://sqs.test/queue"),
			Entries: []types.SendMessageBatchRequestEntry{
				{Id: aws.String("id-1"), MessageBody: aws.String("body")},
			},
		})

		require.NoError(t, err)
		require.Len(t, out.Successful, 1)
		assert.Equal(t, "id-1", aws.ToString(out.Successful[0].Id))
	})
}

func TestProxy_DeleteMessageBatch(t *testing.T) {
	t.Run("it_should_forward_the_call", func(t *testing.T) {
		p := newProxyUnderTest(t)

		out, err := p.DeleteMessageBatch(context.Background(), &awssqs.DeleteMessageBatchInput{
			QueueUrl: aws.String("https://sqs.test/queue"),
			Entries: []types.DeleteMessageBatchRequestEntry{
				{Id: aws.String("id-1"), ReceiptHandle: aws.String("rh-1")},
			},
		})

		require.NoError(t, err)
		require.Len(t, out.Successful, 1)
		assert.Equal(t, "id-1", aws.ToString(out.Successful[0].Id))
	})
}
