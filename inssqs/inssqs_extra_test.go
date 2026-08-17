package inssqs

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/inssqs/sqs"
	"go.uber.org/mock/gomock"
)

// newFakeSQSServer serves the awsjson1.0 SQS wire protocol for the three
// operations this package uses.
func newFakeSQSServer(t *testing.T, getQueueUrlStatus int) (*httptest.Server, *int32) {
	t.Helper()
	var calls int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body, _ := io.ReadAll(r.Body)
		_ = body

		target := r.Header.Get("X-Amz-Target")
		w.Header().Set("Content-Type", "application/x-amz-json-1.0")

		switch {
		case strings.HasSuffix(target, "GetQueueUrl"):
			if getQueueUrlStatus != http.StatusOK {
				w.WriteHeader(getQueueUrlStatus)
				_, _ = w.Write([]byte(`{"__type":"com.amazonaws.sqs#QueueDoesNotExist","message":"no such queue"}`))
				return
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"QueueUrl": "https://sqs.test/queue"})
		case strings.HasSuffix(target, "SendMessageBatch"):
			_, _ = w.Write([]byte(`{"Successful":[{"Id":"test-id","MessageId":"m-1","MD5OfMessageBody":"841a2d689ad86bd1611447453c22c6fc"}],"Failed":[]}`))
		case strings.HasSuffix(target, "DeleteMessageBatch"):
			_, _ = w.Write([]byte(`{"Successful":[{"Id":"test-id"}],"Failed":[]}`))
		default:
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	t.Cleanup(ts.Close)

	return ts, &calls
}

func setFakeAWSEnv(t *testing.T) {
	t.Helper()
	t.Setenv("AWS_ACCESS_KEY_ID", "test-access-key")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "test-secret-key")
	t.Setenv("AWS_SESSION_TOKEN", "")
	t.Setenv("AWS_REGION", "eu-west-1")
}

func TestConfig_setDefaults(t *testing.T) {
	t.Run("it_should_fill_all_zero_fields", func(t *testing.T) {
		c := Config{}
		c.setDefaults()

		assert.Equal(t, 10, c.MaxBatchSize)
		assert.Equal(t, 64*1024, c.MaxBatchSizeBytes)
		assert.Equal(t, 1, c.MaxWorkers)
		assert.Equal(t, 3, c.RetryCount)
	})

	t.Run("it_should_keep_explicit_values", func(t *testing.T) {
		c := Config{MaxBatchSize: 5, MaxBatchSizeBytes: 1024, MaxWorkers: 2, RetryCount: 7}
		c.setDefaults()

		assert.Equal(t, Config{MaxBatchSize: 5, MaxBatchSizeBytes: 1024, MaxWorkers: 2, RetryCount: 7}, c)
	})
}

func TestNewSQS(t *testing.T) {
	t.Run("it_should_panic_when_region_missing", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r, "NewSQS must panic without region")
			err, ok := r.(error)
			require.True(t, ok)
			assert.ErrorIs(t, err, ErrRegionNotSet)
		}()
		NewSQS(Config{QueueName: "q"})
	})

	t.Run("it_should_panic_when_queue_name_missing", func(t *testing.T) {
		defer func() {
			r := recover()
			require.NotNil(t, r, "NewSQS must panic without queue name")
			err, ok := r.(error)
			require.True(t, ok)
			assert.ErrorIs(t, err, ErrQueueNameNotSet)
		}()
		NewSQS(Config{Region: "eu-west-1"})
	})

	t.Run("it_should_panic_when_aws_config_loading_fails", func(t *testing.T) {
		setFakeAWSEnv(t)
		// An invalid boolean makes config.LoadDefaultConfig return an error.
		t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "definitely-not-a-bool")

		defer func() {
			r := recover()
			require.NotNil(t, r, "NewSQS must panic when LoadDefaultConfig fails")
			assert.Contains(t, r.(error).Error(), "error while loading aws sqs config")
		}()
		NewSQS(Config{Region: "eu-west-1", QueueName: "q"})
	})

	t.Run("it_should_build_queue_with_nop_logger_by_default", func(t *testing.T) {
		setFakeAWSEnv(t)
		ts, _ := newFakeSQSServer(t, http.StatusOK)

		q := NewSQS(Config{
			Region:      "eu-west-1",
			QueueName:   "q",
			EndpointUrl: ts.URL,
		})

		require.NotNil(t, q)
		impl, ok := q.(*queue)
		require.True(t, ok)
		assert.Equal(t, "https://sqs.test/queue", aws.ToString(impl.url))
	})

	t.Run("it_should_build_queue_with_leveled_logger_when_log_level_set", func(t *testing.T) {
		setFakeAWSEnv(t)
		ts, _ := newFakeSQSServer(t, http.StatusOK)

		q := NewSQS(Config{
			Region:      "eu-west-1",
			QueueName:   "q",
			EndpointUrl: ts.URL,
			LogLevel:    "ERROR",
		})

		require.NotNil(t, q)
	})

	t.Run("it_should_panic_when_queue_url_cannot_be_resolved", func(t *testing.T) {
		setFakeAWSEnv(t)
		// 400 QueueDoesNotExist is terminal for the SDK, so only the package's
		// own retry loop spins (fast) before the constructor panics.
		ts, _ := newFakeSQSServer(t, http.StatusBadRequest)

		defer func() {
			r := recover()
			require.NotNil(t, r, "NewSQS must panic when GetQueueUrl keeps failing")
			assert.Contains(t, r.(error).Error(), "error while getting queue url")
		}()
		NewSQS(Config{
			Region:      "eu-west-1",
			QueueName:   "q",
			EndpointUrl: ts.URL,
			RetryCount:  2,
		})
	})
}

func TestQueue_sendMessageBatch_edgeCases(t *testing.T) {
	t.Run("it_should_return_nil_for_empty_entries", func(t *testing.T) {
		q, _ := newQueue(t)

		failed, err := q.sendMessageBatch(nil, 3)

		assert.Nil(t, failed)
		assert.NoError(t, err)
	})

	t.Run("it_should_send_nothing_for_empty_batch_list", func(t *testing.T) {
		// No EXPECT on the mock: an empty entry slice must not reach SQS.
		q, _ := newQueue(t)

		failed, err := q.SendMessageBatch(nil)

		assert.Nil(t, failed)
		assert.NoError(t, err)
	})
}

func TestQueue_deleteMessageBatch_edgeCases(t *testing.T) {
	t.Run("it_should_return_nil_for_empty_entries", func(t *testing.T) {
		q, _ := newQueue(t)

		failed, err := q.deleteMessageBatch(nil, 3)

		assert.Nil(t, failed)
		assert.NoError(t, err)
	})

	t.Run("it_should_retry_and_give_up_on_client_error", func(t *testing.T) {
		q, client := newQueue(t)

		client.EXPECT().
			DeleteMessageBatch(gomock.Any(), gomock.Any(), gomock.Any()).
			Times(4).
			Return(nil, assert.AnError)

		failed, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{
			{Id: aws.String("test-id")},
		})

		assert.ErrorIs(t, err, ErrRetryCountExceeded)
		require.Len(t, failed, 1)
		assert.Equal(t, "test-id", aws.ToString(failed[0].Id))
	})
}

func Test_getRequestAttemptCount_withRealMetadata(t *testing.T) {
	t.Run("it_should_count_attempts_from_sdk_metadata", func(t *testing.T) {
		setFakeAWSEnv(t)
		ts, _ := newFakeSQSServer(t, http.StatusOK)

		client := awssqs.New(awssqs.Options{
			Region:       "eu-west-1",
			BaseEndpoint: aws.String(ts.URL),
			Credentials:  credentials.NewStaticCredentialsProvider("test", "test", ""),
		})

		q := queue{
			client:     sqs.NewSQSProxy(client),
			name:       "q",
			url:        aws.String(ts.URL),
			retryCount: 3,
			logger:     inslogger.NewNopLogger(),
		}

		failed, err := q.sendMessageBatch([]SQSMessageEntry{
			{Id: aws.String("test-id"), MessageBody: aws.String("body")},
		}, 3)

		assert.NoError(t, err)
		assert.Nil(t, failed)
	})
}

func TestFakeQueue_SendMessageBatch(t *testing.T) {
	t.Run("it_should_append_entries_and_report_no_failures", func(t *testing.T) {
		q := &FakeQueue{}

		failed, err := q.SendMessageBatch([]SQSMessageEntry{
			{Id: aws.String("a")},
			{Id: aws.String("b")},
		})

		assert.NoError(t, err)
		assert.Nil(t, failed)
		assert.Len(t, q.Data, 2)
	})
}

func TestFakeQueue_DeleteMessageBatch(t *testing.T) {
	t.Run("it_should_remove_entry_with_same_id_pointer", func(t *testing.T) {
		keep := aws.String("keep")
		drop := aws.String("drop")
		q := &FakeQueue{Data: []SQSMessageEntry{{Id: keep}, {Id: drop}}}

		failed, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{{Id: drop}})

		assert.NoError(t, err)
		assert.Nil(t, failed)
		require.Len(t, q.Data, 1)
		assert.Same(t, keep, q.Data[0].Id)
	})

	t.Run("it_should_keep_entry_when_id_value_matches_but_pointer_differs", func(t *testing.T) {
		// PA-39500: pins current (buggy) behavior — see bug registry.
		// DeleteMessageBatch compares *string pointers, not values, so an
		// equal Id held in a different pointer is NOT removed.
		q := &FakeQueue{Data: []SQSMessageEntry{{Id: aws.String("same")}}}

		_, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{{Id: aws.String("same")}})

		assert.NoError(t, err)
		require.Len(t, q.Data, 1, "current behavior: value-equal Id in a new pointer is not matched")
	})

	t.Run("it_should_duplicate_survivors_when_multiple_delete_entries_given", func(t *testing.T) {
		// PA-39500: pins current (buggy) behavior — see bug registry.
		// The nested loop appends a surviving entry once per non-matching
		// delete entry, so two delete entries duplicate every survivor.
		survivor := aws.String("survivor")
		q := &FakeQueue{Data: []SQSMessageEntry{{Id: survivor}}}

		_, err := q.DeleteMessageBatch([]SQSDeleteMessageEntry{
			{Id: aws.String("x")},
			{Id: aws.String("y")},
		})

		assert.NoError(t, err)
		assert.Len(t, q.Data, 2, "current behavior: survivor duplicated once per delete entry")
	})
}
