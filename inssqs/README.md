# inssqs - AWS SQS Package

The `inssqs` package provides an interface and implementation for interacting with Amazon Simple Queue Service (SQS) in Go. It simplifies sending and deleting messages in batches while handling retries, respecting batch size constraints, and supporting concurrent operations.

## Features

- **Batch Operations**: Send and delete messages in batches to SQS queues.
- **Retries**: Automatic retries for failed operations based on a configurable retry count.
- **Concurrency**: Concurrent processing of multiple batches with specified worker count.
- **Error Handling**: Detailed error logging and handling for failed operations.

## Installation

To use this package in your Go project, import it as follows:

```go
import "github.com/useinsider/go-pkg/inssqs"
```

## Usage
### Initialization
Initialize an SQS queue instance by providing configuration settings using `NewSQS()`
```go
config := inssqs.Config{
    Region:            "your-aws-region",
    QueueName:         "your-queue-name",
    RetryCount:        3,
    MaxBatchSize:      10,
    MaxBatchSizeBytes: 1024,
    MaxWorkers:        5,
    LogLevel:          "info",
}

sqs := inssqs.NewSQS(config)
```
### Sending Messages
Send a batch of messages to an SQS queue using `SendMessageBatch()`

```go
messages := []inssqs.SQSMessageEntry{
    // Create SQS message entries here
}

failedMessages, err := sqs.SendMessageBatch(messages)
if err != nil {
    // Handle error
}
```

### Deleting Messages
Delete a batch of messages from an SQS queue using `DeleteMessageBatch()`

```go
messages := []inssqs.SQSDeleteMessageEntry{
    // Create SQS message entries here
}

failedMessages, err := sqs.DeleteMessageBatch(messages)
if err != nil {
    // Handle error
}
```

### Receiving Messages
Receive batch of messages from SQS queue using `ReceiveMessageBatch()`

## Configuration Options
- Region: AWS region where the SQS queue resides.
- QueueName: Name of the SQS queue.
- RetryCount: Number of retry attempts allowed for queue operations.
- MaxBatchSize: Maximum size of a message batch.
- MaxBatchSizeBytes: Maximum size of a message batch in bytes.
- MaxWorkers: Maximum number of workers for concurrent operations.
- LogLevel: Log level for SQS operations.
For more details on each function and its parameters, refer to the code documentation.


## Contribution
Feel free to contribute by forking this repository and creating pull requests. Please ensure to adhere to the existing code style and conventions.






