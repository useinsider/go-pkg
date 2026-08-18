package inskinesis

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func newTestStream(kc KinesisInterface, logBufferSize, maxGroup int) *stream {
	return &stream{
		region:        "eu-west-1",
		name:          "test-stream",
		partitioner:   PartitionerPointer(fakePartitioner),
		kinesisClient: kc,

		logBufferSize:          logBufferSize,
		maxStreamBatchSize:     100,
		maxStreamBatchByteSize: 1 << 16,
		maxGroup:               maxGroup,

		wgLogChan:        &sync.WaitGroup{},
		wgBatchChan:      &sync.WaitGroup{},
		logChannel:       make(chan interface{}, 2000),
		batchChannel:     make(chan []interface{}, 100),
		errChannel:       make(chan error, errorChannelSize),
		stopChannel:      make(chan bool, 10),
		stopBatchChannel: make(chan bool, 10),

		retryCount:    1,
		retryWaitTime: time.Millisecond,

		verbose: true,
	}
}

func successPutOutput() *kinesis.PutRecordsOutput {
	return &kinesis.PutRecordsOutput{
		FailedRecordCount: aws.Int64(0),
		Records:           []*kinesis.PutRecordsResultEntry{},
	}
}

func TestNewKinesis(t *testing.T) {
	t.Run("it_should_require_region", func(t *testing.T) {
		s, err := NewKinesis(Config{StreamName: "s"})
		assert.Nil(t, s)
		assert.EqualError(t, err, "region is required")
	})

	t.Run("it_should_require_stream_name", func(t *testing.T) {
		s, err := NewKinesis(Config{Region: "eu-west-1"})
		assert.Nil(t, s)
		assert.EqualError(t, err, "stream name is required")
	})

	t.Run("it_should_apply_defaults_for_zero_config", func(t *testing.T) {
		si, err := NewKinesis(Config{Region: "eu-west-1", StreamName: "test"})
		require.NoError(t, err)
		require.NotNil(t, si)

		s, ok := si.(*stream)
		require.True(t, ok, "NewKinesis should return *stream")

		assert.Equal(t, 500, s.logBufferSize)
		assert.Equal(t, 100, s.maxStreamBatchSize)
		assert.Equal(t, 1<<16, s.maxStreamBatchByteSize)
		assert.Equal(t, 1, s.maxGroup)
		assert.NotNil(t, s.partitioner)
		assert.Equal(t, 100*time.Millisecond, s.retryWaitTime)

		si.FlushAndStopStreaming()
	})
}

func Test_kinesisProxy_PutRecords(t *testing.T) {
	t.Run("it_should_forward_to_the_kinesis_client", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/x-amz-json-1.1")
			_, _ = w.Write([]byte(`{"FailedRecordCount":0,"Records":[]}`))
		}))
		defer ts.Close()

		sess := session.Must(session.NewSession(&aws.Config{
			Region:      aws.String("eu-west-1"),
			Endpoint:    aws.String(ts.URL),
			DisableSSL:  aws.Bool(true),
			Credentials: credentials.NewStaticCredentials("test", "test", ""),
			MaxRetries:  aws.Int(0),
		}))

		p := &kinesisProxy{kinesis.New(sess)}
		out, err := p.PutRecords(&kinesis.PutRecordsInput{
			StreamName: aws.String("test"),
			Records: []*kinesis.PutRecordsRequestEntry{
				{Data: []byte("r\n"), PartitionKey: aws.String("pk")},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, int64(0), aws.Int64Value(out.FailedRecordCount))
	})
}

func TestStream_PutAndFlush(t *testing.T) {
	t.Run("it_should_flush_pending_buffer_on_stop", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 100, 1)
		s.start()

		mockKinesis.EXPECT().
			PutRecords(gomock.Any()).
			Times(1).
			Return(successPutOutput(), nil)

		s.Put(map[string]string{"k1": "v1"})
		s.Put(map[string]string{"k2": "v2"})
		s.FlushAndStopStreaming()

		assert.Equal(t, 2, s.totalCount)
		assert.Equal(t, 0, s.failedCount)
	})

	t.Run("it_should_flush_mid_stream_when_buffer_size_exceeded", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 2, 1)
		s.start()

		mockKinesis.EXPECT().
			PutRecords(gomock.Any()).
			MinTimes(1).
			Return(successPutOutput(), nil)

		for i := 0; i < 4; i++ {
			s.Put(map[string]int{"i": i})
		}
		s.FlushAndStopStreaming()

		assert.Equal(t, 4, s.totalCount)
	})

	t.Run("it_should_stop_cleanly_with_empty_buffer", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 100, 1)
		s.start()

		s.FlushAndStopStreaming()

		assert.Equal(t, 0, s.totalCount)
	})

	t.Run("it_should_report_batching_error_for_unmarshalable_records", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 1, 1)
		s.start()

		s.Put(make(chan int))
		s.Put(make(chan int))

		select {
		case err := <-s.Error():
			assert.Error(t, err)
		case <-time.After(2 * time.Second):
			t.Fatal("no error received from Error() channel")
		}
	})

	t.Run("it_should_forward_send_errors_to_error_channel", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 100, 1)
		s.start()

		mockKinesis.EXPECT().
			PutRecords(gomock.Any()).
			MinTimes(1).
			Return(nil, errors.New("kinesis unavailable"))

		s.Put(map[string]string{"k": "v"})
		s.FlushAndStopStreaming()

		select {
		case err := <-s.Error():
			assert.ErrorContains(t, err, "kinesis unavailable")
		case <-time.After(2 * time.Second):
			t.Fatal("no error received from Error() channel")
		}
		assert.Equal(t, 1, s.failedCount)
	})
}

func TestStream_PutRecords(t *testing.T) {
	t.Run("it_should_send_transformed_batch", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 100, 1)

		mockKinesis.EXPECT().
			PutRecords(gomock.Any()).
			Times(1).
			Return(successPutOutput(), nil)

		failed, err := s.PutRecords([]interface{}{map[string]string{"k": "v"}})
		assert.NoError(t, err)
		assert.Equal(t, 0, failed)
	})

	t.Run("it_should_return_batch_size_when_transform_fails_entirely", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		s := newTestStream(NewMockKinesisInterface(ctrl), 100, 1)

		batch := []interface{}{make(chan int)}
		failed, err := s.PutRecords(batch)

		assert.Error(t, err)
		assert.Equal(t, len(batch), failed)
	})
}

func Test_putRecords_retryExceeded(t *testing.T) {
	t.Run("it_should_stop_when_retry_count_negative", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		mockKinesis := NewMockKinesisInterface(ctrl)
		s := newTestStream(mockKinesis, 100, 1)

		failed, err := s.putRecords([]*kinesis.PutRecordsRequestEntry{
			{Data: []byte("r\n"), PartitionKey: aws.String(testPartition)},
		}, -1)

		assert.EqualError(t, err, "retry count exceeded")
		assert.Equal(t, 1, failed)
	})
}

func Test_transformRecords_partialFailure(t *testing.T) {
	t.Run("it_should_silently_drop_failed_record_when_later_record_succeeds", func(t *testing.T) {
		s := newTestStream(nil, 100, 1)

		records := []interface{}{
			make(chan int),
			map[string]string{"k": "v"},
		}

		transformed, err := s.transformRecords(records)

		assert.NoError(t, err, "current behavior: error swallowed by later success")
		assert.Len(t, transformed, 1, "current behavior: failed record silently dropped")
	})
}

func Test_addOutputSeparatorIfNeeded(t *testing.T) {
	t.Run("it_should_return_empty_record_unchanged", func(t *testing.T) {
		assert.Empty(t, addOutputSeparatorIfNeeded([]byte{}))
	})

	t.Run("it_should_keep_existing_separator", func(t *testing.T) {
		assert.Equal(t, []byte("x\n"), addOutputSeparatorIfNeeded([]byte("x\n")))
	})

	t.Run("it_should_append_separator_when_missing", func(t *testing.T) {
		assert.Equal(t, []byte("x\n"), addOutputSeparatorIfNeeded([]byte("x")))
	})
}

func TestTakeSliceArg(t *testing.T) {
	t.Run("it_should_convert_typed_slice", func(t *testing.T) {
		out, ok := TakeSliceArg([]string{"a", "b"})
		assert.True(t, ok)
		assert.Equal(t, []interface{}{"a", "b"}, out)
	})

	t.Run("it_should_reject_non_slice_values", func(t *testing.T) {
		out, ok := TakeSliceArg(42)
		assert.False(t, ok)
		assert.Nil(t, out)
	})
}

func Test_createBatches_invalidInput(t *testing.T) {
	t.Run("it_should_reject_non_slice_input", func(t *testing.T) {
		batches, err := createBatches("not-a-slice", 10, 1024)
		assert.EqualError(t, err, "invalid input")
		assert.Nil(t, batches)
	})
}

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "i/o timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

func TestCustomRetryer_ShouldRetry(t *testing.T) {
	retryer := CustomRetryer{Retryer: client.DefaultRetryer{NumMaxRetries: 3}}

	t.Run("it_should_retry_on_net_timeout", func(t *testing.T) {
		req := &request.Request{Error: timeoutNetError{}}
		assert.True(t, retryer.ShouldRetry(req))
	})

	t.Run("it_should_retry_on_connection_reset", func(t *testing.T) {
		req := &request.Request{Error: &net.OpError{Op: "read", Err: errors.New("read: connection reset by peer")}}
		assert.True(t, retryer.ShouldRetry(req))
	})

	t.Run("it_should_delegate_other_errors_to_default_retryer", func(t *testing.T) {
		req := &request.Request{
			Error:     errors.New("some other failure"),
			Retryable: aws.Bool(false),
		}
		assert.False(t, retryer.ShouldRetry(req))
	})
}

func TestPartitioners_UUID(t *testing.T) {
	t.Run("it_should_generate_distinct_uuids", func(t *testing.T) {
		a := Partitioners.UUID(nil)
		b := Partitioners.UUID(nil)
		assert.NotEmpty(t, a)
		assert.NotEqual(t, a, b)
	})
}

func TestPartitionerPointer(t *testing.T) {
	t.Run("it_should_return_callable_pointer", func(t *testing.T) {
		p := PartitionerPointer(fakePartitioner)
		assert.NotNil(t, p)
		assert.Equal(t, testPartition, (*p)("anything"))
	})
}

func TestFakeStream(t *testing.T) {
	t.Run("it_should_record_marshaled_records_and_delegate", func(t *testing.T) {
		inner := &FakeStream{}
		s := &FakeStream{Stream: inner}

		s.Put(map[string]string{"k": "v"})

		require.Len(t, s.Data, 1)
		assert.Equal(t, `{"k":"v"}`, s.Data[0])
		require.Len(t, inner.Data, 1, "Put should delegate to the wrapped stream")
	})

	t.Run("it_should_append_empty_string_on_marshal_error", func(t *testing.T) {
		s := &FakeStream{}
		s.Put(make(chan int))

		require.Len(t, s.Data, 1)
		assert.Equal(t, "", s.Data[0])
	})

	t.Run("it_should_do_nothing_on_Get", func(t *testing.T) {
		(&FakeStream{}).Get()
	})

	t.Run("it_should_return_datum_by_index", func(t *testing.T) {
		s := &FakeStream{}
		s.Put(map[string]string{"a": "1"})
		s.Put(map[string]string{"b": "2"})

		assert.Equal(t, `{"a":"1"}`, s.Datum(0, nil))
	})

	t.Run("it_should_support_negative_index_and_unmarshal", func(t *testing.T) {
		s := &FakeStream{}
		s.Put(map[string]string{"a": "1"})
		s.Put(map[string]string{"b": "2"})

		var target map[string]string
		js := s.Datum(-1, &target)

		assert.Equal(t, `{"b":"2"}`, js)
		assert.Equal(t, map[string]string{"b": "2"}, target)
	})
}
