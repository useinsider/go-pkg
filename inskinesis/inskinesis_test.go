package inskinesis

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kinesis"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
	"testing"
)

func Test_wrapWithPutRecordsRequestEntry(t *testing.T) {
	s := stream{
		partitioner: PartitionerPointer(Partitioners.UUID),
	}

	t.Run("it_should_return_put_records_request_entry_correctly", func(t *testing.T) {
		records := [][]byte{[]byte("record1"), []byte("record2")}
		expected := []*kinesis.PutRecordsRequestEntry{
			{
				Data: []byte("record1\n"),
			},
			{
				Data: []byte("record2\n"),
			},
		}
		actual := s.wrapWithPutRecordsRequestEntry(records)

		assert.Equal(t, expected[0].Data, actual[0].Data)
		assert.Equal(t, expected[1].Data, actual[1].Data)

	})
}

func Test_getFailedRecords(t *testing.T) {
	t.Run("it_should_return_failed_records_correctly", func(t *testing.T) {
		response := &kinesis.PutRecordsOutput{
			Records: []*kinesis.PutRecordsResultEntry{
				{
					ErrorCode: aws.String("error1"),
				},
				{
					ErrorCode: aws.String("error2"),
				},
			},
		}
		records := []*kinesis.PutRecordsRequestEntry{
			{
				Data: []byte("record1\n"),
			},
			{
				Data: []byte("record2\n"),
			},
		}
		expected := [][]byte{[]byte("record1\n"), []byte("record2\n")}
		actual := getFailedRecords(response, records)
		assert.Equal(t, expected, actual)
	})

	t.Run("it_should_return_empty_slice_when_no_failed_records", func(t *testing.T) {
		response := &kinesis.PutRecordsOutput{
			Records: []*kinesis.PutRecordsResultEntry{
				{
					ErrorCode: nil,
				},
				{
					ErrorCode: nil,
				},
			},
		}
		records := []*kinesis.PutRecordsRequestEntry{
			{
				Data: []byte("record1\n"),
			},
			{
				Data: []byte("record2\n"),
			},
		}
		expected := [][]byte{}
		actual := getFailedRecords(response, records)
		assert.Equal(t, expected, actual)
	})
}

func Test_transformRecords(t *testing.T) {
	s := stream{
		partitioner: PartitionerPointer(Partitioners.UUID),
	}

	t.Run("it_should_return_transformed_records_correctly", func(t *testing.T) {
		records := []interface{}{
			map[string]string{
				"key1": "value1",
			},
			map[string]string{
				"key2": "value2",
			},
		}
		expected := []*kinesis.PutRecordsRequestEntry{
			{
				Data: []byte("{\"key1\":\"value1\"}\n"),
			},
			{
				Data: []byte("{\"key2\":\"value2\"}\n"),
			},
		}
		actual, err := s.transformRecords(records)
		assert.Nil(t, err)

		assert.Equal(t, expected[0].Data, actual[0].Data)
		assert.Equal(t, expected[1].Data, actual[1].Data)
	})

	t.Run("it_should_return_error_when_failed_to_transform_records", func(t *testing.T) {
		records := []interface{}{make(chan int)}
		transformed, err := s.transformRecords(records)
		assert.Error(t, err)
		assert.Nil(t, transformed)
	})
}

func Test_CreateBatches(t *testing.T) {
	t.Run("it_should_return_batches_correctly", func(t *testing.T) {
		records := []interface{}{
			map[string]string{
				"key1": "value1",
			},
			map[string]string{
				"key2": "value2",
			},
			map[string]string{
				"key3": "value3",
			},
			map[string]string{
				"key4": "value4",
			},
		}
		expected := [][]interface{}{
			{
				map[string]string{
					"key1": "value1",
				},
				map[string]string{
					"key2": "value2",
				},
			},
			{
				map[string]string{
					"key3": "value3",
				},
				map[string]string{
					"key4": "value4",
				},
			},
		}
		actual, err := createBatches(records, 2, 100)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual)
	})

	t.Run("it_should_return_error_when_failed_to_create_batches", func(t *testing.T) {
		records := []interface{}{make(chan int)}
		batches, err := createBatches(records, 2, 100)
		assert.Error(t, err)
		assert.Nil(t, batches)
	})
}

var testPartition = "test-partition"

func fakePartitioner(_ interface{}) string {
	return testPartition
}

func Test_putRecords(t *testing.T) {
	s := stream{
		name:          "test-stream",
		partitioner:   PartitionerPointer(fakePartitioner),
		kinesisClient: NewMockKinesisInterface(gomock.NewController(t)),
	}

	t.Run("it_should_retry", func(t *testing.T) {
		records := []*kinesis.PutRecordsRequestEntry{
			{
				Data:         []byte("record1\n"),
				PartitionKey: aws.String(testPartition),
			},
			{
				Data:         []byte("record2\n"),
				PartitionKey: aws.String(testPartition),
			},
		}

		resp := kinesis.PutRecordsOutput{
			FailedRecordCount: aws.Int64(2),
			Records: []*kinesis.PutRecordsResultEntry{
				{
					ErrorCode: aws.String("error1"),
				},
				{
					ErrorCode: aws.String("error2"),
				},
			},
		}

		s.kinesisClient.(*MockKinesisInterface).EXPECT().PutRecords(&kinesis.PutRecordsInput{
			Records:    records,
			StreamName: aws.String(s.name),
		}).Times(3).Return(&resp, nil)
		failedCount, _ := s.putRecords(records, 3)

		assert.Equal(t, 2, failedCount)
	})

	t.Run("it_should_not_retry_when_failed_record_count_is_zero", func(t *testing.T) {
		records := []*kinesis.PutRecordsRequestEntry{
			{
				Data:         []byte("record1\n"),
				PartitionKey: aws.String(testPartition),
			},
			{
				Data:         []byte("record2\n"),
				PartitionKey: aws.String(testPartition),
			},
		}

		resp := kinesis.PutRecordsOutput{
			FailedRecordCount: aws.Int64(0),
			Records: []*kinesis.PutRecordsResultEntry{
				{
					ErrorCode: nil,
				},
				{
					ErrorCode: nil,
				},
			},
		}

		s.kinesisClient.(*MockKinesisInterface).EXPECT().PutRecords(&kinesis.PutRecordsInput{
			Records:    records,
			StreamName: aws.String(s.name),
		}).Times(1).Return(&resp, nil)
		failedCount, _ := s.putRecords(records, 3)

		assert.Equal(t, 0, failedCount)
	})
}
