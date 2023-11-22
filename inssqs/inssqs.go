package inssqs

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/pkg/errors"
	"github.com/useinsider/go-pkg/insdash"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/inssqs/sqs"
	"sync"
)

type Interface interface {
	SendMessageBatch(entries []SQSMessageEntry) (failed []SQSMessageEntry, err error)
	DeleteMessageBatch(entries []SQSDeleteMessageEntry) (failed []SQSDeleteMessageEntry, err error)
}

type queue struct {
	client   sqs.API
	name     string
	url      *string
	endpoint string

	retryCount int

	maxBatchSize      int
	maxBatchSizeBytes int
	workers           int

	logger inslogger.Interface
}

// Config represents the configuration settings required for initializing an SQS queue.
type Config struct {
	Region            string // AWS region where the SQS queue resides.
	QueueName         string // Name of the SQS queue.
	RetryCount        int    // Number of retry attempts allowed for queue operations.
	MaxBatchSize      int    // Maximum size of a message batch.
	MaxBatchSizeBytes int    // Maximum size of a message batch in bytes.
	MaxWorkers        int    // Maximum number of workers for concurrent operations.
	LogLevel          string // Log level for SQS operations.

	EndpointUrl string // Endpoint URL for AWS operations.
}

func NewSQS(config Config) Interface {
	if config.Region == "" {
		panic(ErrRegionNotSet)
	}

	if config.QueueName == "" {
		panic(ErrQueueNameNotSet)
	}

	config.setDefaults()

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.Region),
		awsconfig.WithRetryMaxAttempts(config.RetryCount),
		awsconfig.WithRetryMode(aws.RetryModeAdaptive))
	if err != nil {
		panic(errors.Wrap(err, "error while loading aws sqs config"))
	}

	// set endpoint url if provided
	if config.EndpointUrl != "" {
		cfg.BaseEndpoint = aws.String(config.EndpointUrl)
	}

	var logger inslogger.Interface
	if config.LogLevel == "" {
		logger = inslogger.NewNopLogger()
	} else {
		logger = inslogger.NewLogger(inslogger.LogLevel(config.LogLevel))
	}

	q := &queue{
		client:            sqs.NewSQSProxy(awssqs.NewFromConfig(cfg)),
		name:              config.QueueName,
		retryCount:        config.RetryCount,
		workers:           config.MaxWorkers,
		logger:            logger,
		maxBatchSize:      config.MaxBatchSize,
		maxBatchSizeBytes: config.MaxBatchSizeBytes,
	}

	qUrl, err := q.getQueueUrl()
	if err != nil {
		panic(errors.Wrap(err, "error while getting queue url"))
	}

	q.url = qUrl

	return q
}

func (c *Config) setDefaults() {
	if c.MaxBatchSize == 0 {
		c.MaxBatchSize = 10
	}

	if c.MaxBatchSizeBytes == 0 {
		c.MaxBatchSizeBytes = 64 * 1024
	}

	if c.MaxWorkers == 0 {
		c.MaxWorkers = 1
	}

	if c.RetryCount == 0 {
		c.RetryCount = 3
	}
}

// SendMessageBatch sends a batch of messages to an SQS queue, handling retries and respecting batch size constraints.
//
// Parameters:
// - entries: A slice of SQSMessageEntry representing messages to be sent in batches to the SQS queue.
//
// Returns:
// - failedEntries: A slice of SQSMessageEntry containing the messages that failed to be sent after all attempts.
// - err: An error indicating any failure during the sending process, nil if all messages were sent successfully.
//
// Note:
// SendMessageBatch operation has an inherent concurrency limit.
// When multiple concurrent calls reach the maximum workers, the total worker count might exceed expectations.
// Consider this while designing applications for optimal performance.
func (q *queue) SendMessageBatch(entries []SQSMessageEntry) ([]SQSMessageEntry, error) {
	// TODO: make concurrency as optional.
	batches, err := insdash.CreateBatches(entries, q.maxBatchSize, q.maxBatchSizeBytes)
	if err != nil {
		return entries, err
	}

	failedEntries, err := q.sendBatchesConcurrently(batches)
	if err != nil {
		q.logger.Errorf("Error sending %d messages to SQS: %v\n", len(failedEntries), err)
		return failedEntries, err
	}

	return failedEntries, nil
}

// DeleteMessageBatch attempts to delete a batch of messages from an SQS queue using the provided entries,
// handling retries on failure and respecting the specified retry count.
//
// Parameters:
// - entries: A slice of SQSDeleteMessageEntry representing messages to be deleted in batches from the SQS queue.
//
// Returns:
// - failedEntries: A slice of SQSDeleteMessageEntry containing the messages that failed to be deleted after all attempts.
// - err: An error indicating any failure during the deletion process, nil if all messages were deleted successfully.
//
// DeleteMessageBatch operation has an inherent concurrency limit.
// When multiple concurrent calls reach the maximum workers, the total worker count might exceed expectations.
// Consider this while designing applications for optimal performance.
func (q *queue) DeleteMessageBatch(entries []SQSDeleteMessageEntry) (failed []SQSDeleteMessageEntry, err error) {
	// TODO: make concurrency as optional.
	batches, err := insdash.CreateBatches(entries, q.maxBatchSize, q.maxBatchSizeBytes)
	if err != nil {
		return entries, err
	}

	failedEntries, err := q.deleteBatchesConcurrently(batches)
	if err != nil {
		q.logger.Errorf("Error deleting %d messages from SQS: %v\n", len(failedEntries), err)
		return failedEntries, err
	}

	return failedEntries, nil
}

// sendBatchesConcurrently concurrently processes multiple batches of send operations
// on an SQS queue using a specified number of workers and retry logic.
//
// Parameters:
// - batches: A slice of slices, each containing SQSMessageEntry representing send operations in batches.
//
// Returns:
// - failedEntries: A slice of SQSMessageEntry containing the failed send operations across all batches.
// - err: An error indicating any failure during the concurrent sending process, nil if all operations succeeded.
func (q *queue) sendBatchesConcurrently(batches [][]SQSMessageEntry) ([]SQSMessageEntry, error) {
	failedEntries, err := doConcurrently(batches, q.workers, q.retryCount, q.sendMessageBatch)

	return failedEntries, err
}

// deleteBatchesConcurrently concurrently processes multiple batches of delete operations
// on an SQS queue using a specified number of workers and retry logic.
//
// Parameters:
// - batches: A slice of slices, each containing SQSDeleteMessageEntry representing delete operations in batches.
//
// Returns:
// - failedEntries: A slice of SQSDeleteMessageEntry containing the failed delete operations across all batches.
// - err: An error indicating any failure during the concurrent deletion process, nil if all operations succeeded.
func (q *queue) deleteBatchesConcurrently(batches [][]SQSDeleteMessageEntry) ([]SQSDeleteMessageEntry, error) {
	failedEntries, err := doConcurrently(batches, q.workers, q.retryCount, q.deleteMessageBatch)
	if err != nil {
		return nil, err
	}

	return failedEntries, err
}

// deleteMessageBatch attempts to delete a batch of messages from an SQS queue, handling retries on failure.
//
// Parameters:
// - entries: A slice of SQSDeleteMessageEntry representing messages to be deleted in batches from the SQS queue.
// - retryCount: An integer indicating the number of retry attempts allowed for deleting the messages.
//
// Returns:
// - failedEntries: A slice of SQSDeleteMessageEntry containing the messages that failed to be deleted after all attempts.
// - err: An error indicating any failure during the deletion process, nil if all messages were deleted successfully.
func (q *queue) deleteMessageBatch(entries []SQSDeleteMessageEntry, retryCount int) ([]SQSDeleteMessageEntry, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	if retryCount == 0 {
		return entries, ErrRetryCountExceeded
	}

	batchEntries := make([]types.DeleteMessageBatchRequestEntry, len(entries))
	for i, e := range entries {
		batchEntries[i] = e.toDeleteMessageBatchRequestEntry()
	}

	batch := &awssqs.DeleteMessageBatchInput{
		Entries:  batchEntries,
		QueueUrl: q.url,
	}

	res, err := q.client.DeleteMessageBatch(context.Background(), batch)
	if err != nil {
		q.logger.Errorf("Error sending %d messages to SQS: %v\n", len(entries), err)
		return q.deleteMessageBatch(entries, retryCount-1)
	}

	attempts := getRequestAttemptCount(res.ResultMetadata)

	if len(res.Failed) == 0 {
		q.logger.Logf("Successfully sent %d messages to SQS after %d attempts\n", len(entries), attempts)

		return nil, nil
	}

	failedEntries := getFailedEntries(entries, res.Failed)
	q.logger.Logf("Failed to send %d messages to SQS after %d attempts\n", len(failedEntries), attempts)

	return q.deleteMessageBatch(failedEntries, retryCount-1)
}

// sendMessageBatch sends a batch of SQS messages in multiple attempts based on batch size and size constraints.
//
// Parameters:
// - entries: A slice of SQSMessageEntry representing messages to be sent in batches to the SQS queue.
//
// Returns:
// - failedEntries: A slice of SQSMessageEntry containing the messages that failed to be sent after all attempts.
// - err: An error indicating any failure during the sending process, nil if all messages were sent successfully.
func (q *queue) sendMessageBatch(entries []SQSMessageEntry, retryCount int) (failed []SQSMessageEntry, err error) {
	if len(entries) == 0 {
		return nil, nil
	}

	if retryCount == 0 {
		return entries, ErrRetryCountExceeded
	}

	batchEntries := make([]types.SendMessageBatchRequestEntry, len(entries))
	for i, e := range entries {
		batchEntries[i] = e.toSendMessageBatchRequestEntry()
	}

	batch := &awssqs.SendMessageBatchInput{
		Entries:  batchEntries,
		QueueUrl: q.url,
	}

	res, err := q.client.SendMessageBatch(context.Background(), batch)
	if err != nil {
		q.logger.Errorf("Error sending %d messages to SQS: %v\n", len(entries), err)
		return q.sendMessageBatch(entries, retryCount-1)
	}

	attempts := getRequestAttemptCount(res.ResultMetadata)

	if len(res.Failed) == 0 {
		q.logger.Logf("Sent %d messages to SQS after %d attempts\n", len(entries), attempts)
		return nil, nil
	}

	failedEntries := getFailedEntries(entries, res.Failed)

	return q.sendMessageBatch(failedEntries, retryCount-1)
}

// getQueueUrl retrieves the URL of an SQS queue based on its name using the provided SQS client.
//
// This function fetches the queue URL if it's not already cached within the 'queue' instance.
//
// Returns:
// - queueUrl: A pointer to a string containing the URL of the SQS queue.
// - err: An error if fetching the queue URL fails, nil otherwise.
func (q *queue) getQueueUrl() (queueUrl *string, err error) {
	if q.url != nil {
		return q.url, nil
	}

	res, err := q.client.GetQueueUrl(context.Background(), &awssqs.GetQueueUrlInput{
		QueueName: aws.String(q.name),
	})
	if err != nil {
		return nil, err
	}

	q.url = res.QueueUrl

	return res.QueueUrl, nil
}

// getFailedEntries retrieves failed elements by matching their IDs from a collection.
// Returns a slice of elements corresponding to the failed IDs, maintaining the original order.
func getFailedEntries[T entry](entries []T, failed []types.BatchResultErrorEntry) []T {
	failedEntries := make([]T, len(failed))
	for i, f := range failed {
		for _, e := range entries {
			if *e.getId() == *f.Id {
				failedEntries[i] = e
			}
		}
	}

	return failedEntries
}

// getRequestAttemptCount retrieves the number of attempts from AWS SDK request metadata.
// Returns the attempt counts or -1 if not found.
func getRequestAttemptCount(metadata middleware.Metadata) int {
	attempts, ok := retry.GetAttemptResults(metadata)
	if !ok {
		return -1
	}

	return len(attempts.Results)
}

// doConcurrently executes a function concurrently across multiple batches of elements of type T,
// managing parallel processing with specified worker count and retry logic.
//
// Parameters:
//   - batches: A slice of slices, each containing elements of type T. Represents the batches
//     of operations to be processed concurrently.
//   - workers: An integer defining the maximum number of goroutines (workers) allowed to run
//     simultaneously for executing the provided function.
//   - retryCount: An integer indicating the number of retry attempts permitted for each batch operation.
//   - f: A function that processes a single batch of elements of type T, given a retry count.
//     It takes two parameters:
//   - A slice of elements of type T to be processed.
//   - An integer representing the retry count for the operation.
//     It returns two values:
//   - A slice of elements of type T that failed during processing.
//   - An error indicating any error occurred during the processing.
//
// Returns:
//   - failedEntries: A slice containing all the failed entries encountered during the concurrent operations
//     across all batches.
//   - outerErr: An error variable that holds any errors encountered during the concurrent execution.
//     It remains nil if no errors occurred during the execution.
func doConcurrently[T any](batches [][]T, workers int, retryCount int, f func([]T, int) ([]T, error)) ([]T, error) {
	if len(batches) == 0 {
		return nil, nil
	}

	concurrentLimiter := make(chan struct{}, workers)
	wg := sync.WaitGroup{}
	var outerErr error
	failedEntriesChan := make(chan []T)

	for _, batch := range batches {
		wg.Add(1)
		go func(b []T) {
			defer wg.Done()
			concurrentLimiter <- struct{}{}
			defer func() { <-concurrentLimiter }()
			fe, err := f(b, retryCount+1)
			if err != nil {
				outerErr = err
			}
			failedEntriesChan <- fe
		}(batch)
	}

	go func() {
		wg.Wait()
		close(failedEntriesChan)
	}()

	var failedEntries []T
	for failedBatch := range failedEntriesChan {
		failedEntries = append(failedEntries, failedBatch...)
	}

	return failedEntries, outerErr
}
