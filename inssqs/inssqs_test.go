package inssqs

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/inssqs/sqs"
	"go.uber.org/mock/gomock"
	"testing"
)

func TestQueue_SendMessageBatch(t *testing.T) {
	t.Run("should_failed_records_as_nil_when_successful", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().SendMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).Return(&awssqs.SendMessageBatchOutput{
			Failed: nil,
			Successful: []types.SendMessageBatchResultEntry{
				{Id: aws.String("test-id")},
			},
			ResultMetadata: middleware.Metadata{},
		}, nil)

		failed, err := q.SendMessageBatch([]SQSMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.Nil(t, err, "err should be nil")
		assert.Nil(t, failed, "failed should be nil")
	})

	t.Run("should_return_failed_records_when_failed", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			SendMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(4).
			Return(&awssqs.SendMessageBatchOutput{
				Failed: []types.BatchResultErrorEntry{
					{Id: aws.String("test-id")},
				},
				Successful:     nil,
				ResultMetadata: middleware.Metadata{},
			}, nil)

		failed, err := q.SendMessageBatch([]SQSMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.NotNil(t, err, "err should not be nil")
		assert.NotNil(t, failed, "failed should not be nil")
		assert.Equal(t, failed[0].Id, aws.String("test-id"), "failed id should be equal to test-id")
	})

	t.Run("should_return_error_when_failed", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			SendMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&awssqs.SendMessageBatchOutput{
				Failed: []types.BatchResultErrorEntry{
					{Id: aws.String("test-id")},
				}}, assert.AnError).
			Times(4)

		failed, err := q.SendMessageBatch([]SQSMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.Error(t, err, "err should not be nil")
		assert.NotNil(t, failed, "failed should not be nil")
		assert.Equal(t, failed[0].Id, aws.String("test-id"), "failed id should be equal to test-id")

	})

}

func TestQueue_DeleteMessageBatch(t *testing.T) {
	t.Run("should_return_nil_when_successful", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			DeleteMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&awssqs.DeleteMessageBatchOutput{
				Failed: nil,
			}, nil)

		failed, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.Nil(t, err, "err should be nil")
		assert.Nil(t, failed, "failed should be nil")
	})

	t.Run("should_return_failed_records_when_failed", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			DeleteMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(4).
			Return(&awssqs.DeleteMessageBatchOutput{
				Failed: []types.BatchResultErrorEntry{
					{Id: aws.String("test-id")},
				},
			}, nil)

		failed, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.Nil(t, err, "err should be nil")
		assert.NotNil(t, failed, "failed should not be nil")
		assert.Equal(t, failed[0].Id, aws.String("test-id"), "failed id should be equal to test-id")
	})
}

func TestQueue_getQueueUrl(t *testing.T) {
	t.Run("should_return_queue_url_when_successful", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			GetQueueUrl(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(&awssqs.GetQueueUrlOutput{
				QueueUrl: aws.String("test-queue-url"),
			}, nil)

		queueUrl, err := q.getQueueUrl()

		assert.Nil(t, err, "err should be nil")
		assert.Equal(t, queueUrl, aws.String("test-queue-url"), "queue url should be equal to test-queue-url")

		t.Run("should_return_queue_url_from_cache_when_called_twice", func(_ *testing.T) {
			client.EXPECT().GetQueueUrl(gomock.Any(), gomock.Any(), gomock.Any()).Times(0) // should not be called

			queueUrl, err := q.getQueueUrl()

			assert.Nil(t, err, "err should be nil")
			assert.Equal(t, queueUrl, aws.String("test-queue-url"), "queue url should be equal to test-queue-url")
		})
	})

	t.Run("should_return_error_when_failed", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			GetQueueUrl(gomock.Any(), gomock.Any(), gomock.Any()).
			Return(nil, assert.AnError)

		queueUrl, err := q.getQueueUrl()

		assert.Error(t, err, "err should not be nil")
		assert.Nil(t, queueUrl, "queue url should be nil")
	})
}

func Test_getFailedEntries(t *testing.T) {
	t.Run("should_return_failed_entries", func(t *testing.T) {
		entries := []SQSMessageEntry{
			{Id: aws.String("test-id-1")},
			{Id: aws.String("test-id-2")},
			{Id: aws.String("test-id-3")},
		}

		failed := []types.BatchResultErrorEntry{
			{Id: aws.String("test-id-1")},
			{Id: aws.String("test-id-3")},
		}

		failedEntries := getFailedEntries(entries, failed)

		assert.Len(t, failedEntries, 2, "failed entries length should be 2")
		assert.Equal(t, failedEntries[0].Id, aws.String("test-id-1"))
		assert.Equal(t, failedEntries[1].Id, aws.String("test-id-3"))
	})
}

func newQueue(t *testing.T) (queue, *sqs.MockAPI) {
	ctrl := gomock.NewController(t)
	client := sqs.NewMockSQS(ctrl)

	return queue{
		client:     client,
		name:       "test-queue",
		retryCount: 3,
		logger:     inslogger.NewLogger(inslogger.Debug),
	}, client
}
