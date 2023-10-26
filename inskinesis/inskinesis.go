package inskinesis

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

const (
	outputSeparator = byte('\n')
	RetryCount      = 3
	RetryWaitTime   = 100 * time.Millisecond
)

type KinesisInterface interface {
	PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error)
}

type kinesisProxy struct {
	*kinesis.Kinesis
}

func (k *kinesisProxy) PutRecords(input *kinesis.PutRecordsInput) (*kinesis.PutRecordsOutput, error) {
	return k.Kinesis.PutRecords(input)
}

// StreamInterface defines the interface for a Kinesis stream.
type StreamInterface interface {
	Put(record interface{})
	Error() <-chan error
	FlushAndStopStreaming()
}

// stream represents a Kinesis stream and its properties.
type stream struct {
	region        string               // AWS region where the Kinesis stream is located.
	name          string               // Name of the Kinesis stream.
	partitioner   *PartitionerFunction // The partitioning function used to determine the partition key for records.
	kinesisClient KinesisInterface     // AWS Kinesis client for interacting with the stream.

	logBufferSize          int // Maximum size of the log buffer for records.
	maxStreamBatchSize     int // Maximum size of each batch of records to be sent to the stream.
	maxStreamBatchByteSize int // Maximum size (in bytes) of each batch of records.
	maxGroup               int // Maximum number of concurrent groups for sending records.

	mu               sync.Mutex         // Mutex to synchronize access to the stream.
	wgLogChan        *sync.WaitGroup    // WaitGroup to manage goroutines.
	wgBatchChan      *sync.WaitGroup    // WaitGroup to manage goroutines.
	logChannel       chan interface{}   // Channel for receiving individual log records.
	batchChannel     chan []interface{} // Channel for sending batches of log records.
	stopChannel      chan bool          // Channel to signal the termination of the streaming process.
	errChannel       chan error         // Channel for receiving errors.
	stopBatchChannel chan bool          // Channel to signal the termination of batch streaming.
	logBuffer        []interface{}      // Buffer for accumulating log records before batching.

	failedCount int // Counter for the number of failed record submissions.
	totalCount  int // Counter for the total number of records sent to the stream.
}

type Config struct {
	Region                 string
	StreamName             string
	Partitioner            *PartitionerFunction
	MaxStreamBatchSize     int
	MaxStreamBatchByteSize int
	MaxBatchSize           int
	MaxGroup               int
}

// NewKinesis creates a new Kinesis stream.
func NewKinesis(config Config) (StreamInterface, error) {
	if config.Region == "" {
		return nil, errors.New("region is required")
	}

	if config.StreamName == "" {
		return nil, errors.New("stream name is required")
	}
	awsConfig := aws.Config{Region: aws.String(config.Region)}
	awsSession, err := session.NewSession(&awsConfig)
	if err != nil {
		return nil, err
	}

	kinesisClient := kinesis.New(awsSession)

	s := &stream{
		region:        config.Region,
		name:          config.StreamName,
		partitioner:   config.Partitioner,
		kinesisClient: kinesisClient,

		logBufferSize:          config.MaxBatchSize,
		maxStreamBatchSize:     config.MaxStreamBatchSize,
		maxStreamBatchByteSize: config.MaxStreamBatchByteSize,
		maxGroup:               config.MaxGroup,

		wgLogChan:        &sync.WaitGroup{},
		wgBatchChan:      &sync.WaitGroup{},
		logChannel:       make(chan interface{}, 1000),
		batchChannel:     make(chan []interface{}, 100),
		stopChannel:      make(chan bool),
		stopBatchChannel: make(chan bool),
	}

	if s.logBufferSize == 0 {
		s.logBufferSize = 500
	}

	if s.maxStreamBatchSize == 0 {
		s.maxStreamBatchSize = 100
	}

	if s.maxStreamBatchByteSize == 0 {
		s.maxStreamBatchByteSize = int(math.Pow(2, 18))
	}

	if s.maxGroup == 0 {
		s.maxGroup = 1
	}

	if s.partitioner == nil {
		s.partitioner = PartitionerPointer(Partitioners.UUID)
	}

	s.start()

	return s, nil
}

// Error returns the channel for receiving errors.
func (s *stream) Error() <-chan error {
	return s.errChannel
}

func (s *stream) startStreaming() {
	for {
		select {
		case record := <-s.logChannel:
			s.mu.Lock()
			s.logBuffer = append(s.logBuffer, record)
			s.mu.Unlock()
			if len(s.logBuffer) > s.logBufferSize {
				batch := s.logBuffer
				s.logBuffer = make([]interface{}, 0)

				batches, err := CreateBatches(batch, s.maxStreamBatchSize, s.maxStreamBatchByteSize)
				if err != nil {
					s.errChannel <- err
					continue
				}

				for _, b := range batches {
					s.wgBatchChan.Add(1)
					s.batchChannel <- b
				}
			}
			s.wgLogChan.Done()
		case <-s.stopChannel:
			s.stopAndWaitBatchStreaming()
			s.wgLogChan.Done()
			return
		}
	}
}

func (s *stream) stopAndWaitBatchStreaming() {
	s.wgBatchChan.Wait()
	s.wgBatchChan.Add(1)
	s.stopBatchChannel <- true
	s.wgBatchChan.Wait()

	close(s.batchChannel)
}

func (s *stream) stopAndWaitLogStreaming() {
	s.wgLogChan.Wait()
	s.wgLogChan.Add(1)
	s.stopChannel <- true
	s.wgLogChan.Wait()

	close(s.logChannel)
}

func (s *stream) startBatchStreaming() {
	concurrentLimiter := make(chan struct{}, s.maxGroup)
	for {
		select {
		case batch := <-s.batchChannel:
			concurrentLimiter <- struct{}{}
			go func() {
				defer func() {
					s.wgBatchChan.Done()
					<-concurrentLimiter
				}()

				s.totalCount += len(batch)
				failedCount, err := s.PutRecords(batch)
				if err != nil {
					fmt.Printf("Error sending records to Kinesis stream %s: %v\n", s.name, err)
					s.errChannel <- err
					return
				}

				fmt.Printf("Sent %d records to Kinesis stream %s\n", len(batch), s.name)

				s.failedCount += failedCount
			}()
		case <-s.stopBatchChannel:
			if len(s.logBuffer) == 0 {
				s.wgBatchChan.Done()
				return
			}

			lastBatch := s.logBuffer
			s.totalCount += len(lastBatch)
			s.logBuffer = make([]interface{}, 0)

			batches, _ := CreateBatches(lastBatch, s.maxStreamBatchSize, s.maxStreamBatchByteSize)

			for _, b := range batches {
				failedCount, err := s.PutRecords(b)
				s.failedCount += failedCount
				if err != nil {
					fmt.Printf("Error sending records to Kinesis stream %s: %v\n", s.name, err)
					s.errChannel <- err
					s.wgBatchChan.Done()
					return
				}

			}

			s.wgBatchChan.Done()
			return
		}
	}
}

func (s *stream) start() {
	go s.startStreaming()
	go s.startBatchStreaming()
}

func (s *stream) FlushAndStopStreaming() {
	s.stopAndWaitLogStreaming()
	close(s.errChannel)
	fmt.Printf("%d/%d records sent to Kinesis stream %s\n", s.totalCount-s.failedCount, s.totalCount, s.name)
}

// PutRecords sends records to the Kinesis stream.
func (s *stream) PutRecords(batch []interface{}) (int, error) {
	transformed, err := s.transformRecords(batch)
	if err != nil {
		return len(batch), err
	}

	failedCount, err := s.putRecords(transformed, RetryCount)

	return failedCount, err
}

// Put sends a single record to the Kinesis stream.
func (s *stream) Put(record interface{}) {
	s.wgLogChan.Add(1)
	s.logChannel <- record
}

func (s *stream) putRecords(batch []*kinesis.PutRecordsRequestEntry, retryCount int) (int, error) {
	fmt.Printf("Sending %d records to Kinesis stream %s\n", len(batch), s.name)
	if retryCount == 0 {
		fmt.Printf("Retry count exceeded for Kinesis stream %s\n", s.name)
		return len(batch), errors.New("retry count exceeded")
	}

	res, err := s.kinesisClient.PutRecords(&kinesis.PutRecordsInput{
		Records:    batch,
		StreamName: aws.String(s.name),
	})

	if err != nil {
		fmt.Printf("Error sending records to Kinesis stream %s: %v\n", s.name, err)
		return len(batch), err
	}

	if res != nil && res.FailedRecordCount != nil && *res.FailedRecordCount > 0 {
		fmt.Printf("Failed to send %d records to Kinesis stream %s\n", *res.FailedRecordCount, s.name)
		failedRecords := s.wrapWithPutRecordsRequestEntry(getFailedRecords(res, batch))
		batch = failedRecords
		retryCount--

		fmt.Printf("Retrying %d records to Kinesis stream %s\n", len(batch), s.name)
		time.Sleep(RetryWaitTime)
		failed, err := s.putRecords(batch, retryCount)
		if err != nil {
			return failed, err
		}
	}
	return 0, err
}

func (s *stream) transformRecords(records []interface{}) ([]*kinesis.PutRecordsRequestEntry, error) {
	var transformedRecords []*kinesis.PutRecordsRequestEntry
	failedRecords := 0
	var err error
	var js []byte
	for _, record := range records {
		js, err = json.Marshal(record)
		if err != nil {
			failedRecords += 1
			continue
		}

		transformedRecords = append(transformedRecords, &kinesis.PutRecordsRequestEntry{
			Data:         append(js, outputSeparator),
			PartitionKey: aws.String((*s.partitioner)(js)),
		})
	}

	if failedRecords > 0 {
		fmt.Printf("Failed to transform %d records to Kinesis stream %s\n", failedRecords, s.name)
	}

	return transformedRecords, err
}

func getFailedRecords(response *kinesis.PutRecordsOutput, records []*kinesis.PutRecordsRequestEntry) [][]byte {
	failedRecords := make([][]byte, 0)

	for i, record := range response.Records {
		if record.ErrorCode != nil {
			failedRecords = append(failedRecords, records[i].Data)
		}
	}

	return failedRecords
}

func (s *stream) wrapWithPutRecordsRequestEntry(records [][]byte) []*kinesis.PutRecordsRequestEntry {
	var transformedRecords []*kinesis.PutRecordsRequestEntry

	for _, record := range records {
		transformedRecords = append(transformedRecords, &kinesis.PutRecordsRequestEntry{
			Data:         append(record, outputSeparator),
			PartitionKey: aws.String((*s.partitioner)(record)),
		})
	}

	return transformedRecords
}

func CreateBatches(v interface{}, recordLimit int, byteLimit int) ([][]interface{}, error) {
	records, ok := TakeSliceArg(v)
	if !ok {
		return nil, errors.New("invalid input")
	}

	var batches = make([][]interface{}, 0)
	buffer := make([]interface{}, 0)
	bufferSize := 0

	for _, record := range records {
		js, jsErr := json.Marshal(record)
		if jsErr != nil {
			return nil, jsErr
		}

		recordSize := len(js)
		sizeExceeds := bufferSize+recordSize > byteLimit
		bufferFull := len(buffer) == recordLimit

		if len(buffer) > 0 && (bufferFull || sizeExceeds) {
			batches = append(batches, buffer)
			buffer = make([]interface{}, 0)
			bufferSize = 0
		}

		buffer = append(buffer, record)
		bufferSize += recordSize
	}

	if len(buffer) > 0 {
		batches = append(batches, buffer)
	}

	return batches, nil
}
