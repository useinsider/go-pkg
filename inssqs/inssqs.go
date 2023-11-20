package inssqs

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/retry"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/aws/smithy-go/middleware"
	"github.com/useinsider/go-pkg/inslogger"
	"github.com/useinsider/go-pkg/insparallel"
	"github.com/useinsider/go-pkg/inssqs/sqs"
)

type Interface interface {
	SendMessageBatch(entries []SQSMessageEntry) (failed []SQSMessageEntry, err error)
	DeleteMessageBatch(entries []SQSDeleteMessageEntry) (failed []SQSDeleteMessageEntry, err error)
}

type queue struct {
	client sqs.API

	name       string
	retryCount int
	url        *string

	logger inslogger.AppLogger
}

type Config struct {
	Region    string
	QueueName string

	RetryCount int

	LogLevel string
}

func NewSQS(config Config) Interface {
	envConfig, err := awsconfig.NewEnvConfig()
	if err != nil {
		return nil
	}

	cfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(config.Region),
		awsconfig.WithSharedConfigProfile(envConfig.SharedConfigProfile),
		awsconfig.WithRetryMaxAttempts(config.RetryCount),
		awsconfig.WithRetryMode(aws.RetryModeAdaptive),
	)
	if err != nil {
		return nil
	}

	logger := inslogger.NewLogger(inslogger.Info)
	if config.LogLevel != "" {
		logger.SetLevel(inslogger.LogLevel(config.LogLevel))
	}

	q := &queue{
		client: sqs.NewSQSProxy(awssqs.NewFromConfig(cfg)),
		name:   config.QueueName,

		retryCount: config.RetryCount,
	}

	qUrl, err := q.getQueueUrl()
	if err != nil {
		panic(err)
	}

	q.url = qUrl

	return q
}
func (q *queue) SendMessageBatch(entries []SQSMessageEntry) ([]SQSMessageEntry, error) {
	batches, err := createBatches(entries, 10, 256*1024*1024)
	if err != nil {
		return entries, err
	}

	failedRecords, err := q.sendBatchesConcurrently(batches)
	if err != nil {
		q.logger.Logf("Error sending %d messages to SQS: %v\n", len(failedRecords), err)
		return failedRecords, err
	}

	return failedRecords, nil
}

func (q *queue) DeleteMessageBatch(entries []SQSDeleteMessageEntry) (failed []SQSDeleteMessageEntry, err error) {
	batches, err := createBatches(entries, 10, 256*1024*1024)
	if err != nil {
		return entries, err
	}

	failedRecords, err := q.deleteBatchesConcurrently(batches)
	if err != nil {
		q.logger.Logf("Error deleting %d messages from SQS: %v\n", len(failedRecords), err)
		return failedRecords, err
	}

	return failedRecords, nil
}

func (q *queue) sendBatchesConcurrently(batches [][]SQSMessageEntry) ([]SQSMessageEntry, error) {
	wrkGrp := insparallel.NewWorkGroup[[]SQSMessageEntry, []SQSMessageEntry](5)

	for _, batch := range batches {
		b := batch
		wrkGrp.Add(func([]SQSMessageEntry) ([]SQSMessageEntry, error) {
			return q.sendMessageBatch(b, q.retryCount+1)
		}, b)
	}

	wrkGrp.Run()
	wrkGrp.Wait()

	var failedRecords []SQSMessageEntry
	var err error
	for _, work := range wrkGrp.Works {
		if work.Err != nil {
			err = work.Err
			q.logger.Logf("Error sending %d messages to SQS: %v\n", len(work.Params), work.Err)
		}

		if work.RetVal != nil {
			failedRecords = append(failedRecords, work.RetVal...)
		}
	}

	return failedRecords, err
}

func (q *queue) deleteBatchesConcurrently(batches [][]SQSDeleteMessageEntry) ([]SQSDeleteMessageEntry, error) {
	wrkGrp := insparallel.NewWorkGroup[[]SQSDeleteMessageEntry, []SQSDeleteMessageEntry](5)

	for _, batch := range batches {
		b := batch
		wrkGrp.Add(func([]SQSDeleteMessageEntry) ([]SQSDeleteMessageEntry, error) {
			return q.deleteMessageBatch(b, q.retryCount+1)
		}, b)
	}

	wrkGrp.Run()
	wrkGrp.Wait()

	var failedRecords []SQSDeleteMessageEntry
	var err error
	for _, work := range wrkGrp.Works {
		if work.Err != nil {
			err = work.Err
			q.logger.Logf("Error deleting %d messages from SQS: %v\n", len(work.Params), work.Err)
		}

		if work.RetVal != nil {
			failedRecords = append(failedRecords, work.RetVal...)
		}
	}

	return failedRecords, err
}

func (q *queue) deleteMessageBatch(entries []SQSDeleteMessageEntry, retryCount int) ([]SQSDeleteMessageEntry, error) {
	if len(entries) == 0 {
		return nil, nil
	}

	if retryCount == 0 {
		return entries, nil
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
		q.logger.Logf("Error sending %d messages to SQS: %v\n", len(entries), err)
		return entries, err
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

func (q *queue) sendMessageBatch(entries []SQSMessageEntry, retryCount int) (failed []SQSMessageEntry, err error) {
	if len(entries) == 0 {
		return nil, nil
	}

	if retryCount == 0 {
		return entries, nil
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
		q.logger.Logf("Error sending %d messages to SQS: %v\n", len(entries), err)
		return entries, err
	}

	attempts := getRequestAttemptCount(res.ResultMetadata)

	if len(res.Failed) == 0 {
		q.logger.Logf("Successfully sent %d messages to SQS after %d attempts\n", len(entries), attempts)

		return nil, nil
	}

	failedEntries := getFailedEntries(entries, res.Failed)
	q.logger.Logf("Failed to send %d messages to SQS after %d attempts\n", len(failedEntries), attempts)

	return q.sendMessageBatch(failedEntries, retryCount-1)
}

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

func getRequestAttemptCount(metadata middleware.Metadata) int {
	attempts, ok := retry.GetAttemptResults(metadata)
	if !ok {
		return -1
	}

	return len(attempts.Results)
}
